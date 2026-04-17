package pxe

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/krolaw/dhcp4"
	"github.com/krolaw/dhcp4/conn"
)

// DHCPServer implements a DHCP server for PXE boot
type DHCPServer struct {
	config       *DHCPConfig
	leases       map[string]net.IP
	reservations map[string]net.IP // MAC -> IP
	mu           sync.RWMutex
	conn         *conn.UDPConn
	handler      *DHCPHandler
}

// DHCPConfig contains DHCP server configuration
type DHCPConfig struct {
	Interface    string
	ServerIP     string
	RangeStart   string
	RangeEnd     string
	Subnet       string
	Router       string
	DNSServer    string
	TFTPServer   string
	BIOSBootFile string
	UEFIBootFile string
	LeaseTime    time.Duration
}

// NewDHCPServer creates a new DHCP server
func NewDHCPServer(config *DHCPConfig) (*DHCPServer, error) {
	if config.LeaseTime == 0 {
		config.LeaseTime = 24 * time.Hour
	}

	server := &DHCPServer{
		config:       config,
		leases:       make(map[string]net.IP),
		reservations: make(map[string]net.IP),
	}

	server.handler = &DHCPHandler{server: server}

	return server, nil
}

// Start starts the DHCP server
func (s *DHCPServer) Start() error {
	// Listen on DHCP port (67)
	pc, err := conn.NewUDP4BoundListener(s.config.Interface, ":67")
	if err != nil {
		return fmt.Errorf("failed to bind DHCP port: %w", err)
	}

	s.conn = pc

	go func() {
		err := dhcp4.Serve(pc, s.handler)
		if err != nil {
			fmt.Printf("DHCP server error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the DHCP server
func (s *DHCPServer) Stop() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// AddReservation adds a static IP reservation for a MAC address
func (s *DHCPServer) AddReservation(mac net.HardwareAddr, ip net.IP, hostname string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reservations[mac.String()] = ip
}

// RemoveReservation removes a static IP reservation
func (s *DHCPServer) RemoveReservation(mac net.HardwareAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.reservations, mac.String())
}

// GetReservation checks if there's a reservation for a MAC address
func (s *DHCPServer) GetReservation(mac net.HardwareAddr) (net.IP, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ip, ok := s.reservations[mac.String()]
	return ip, ok
}

// DHCPHandler handles DHCP requests
type DHCPHandler struct {
	server *DHCPServer
}

// ServeDHCP implements dhcp4.Handler interface
func (h *DHCPHandler) ServeDHCP(packet dhcp4.Packet, msgType dhcp4.MessageType, options dhcp4.Options) dhcp4.Packet {
	switch msgType {
	case dhcp4.Discover:
		return h.handleDiscover(packet, options)
	case dhcp4.Request:
		return h.handleRequest(packet, options)
	case dhcp4.Release:
		h.handleRelease(packet)
		return nil
	default:
		return nil
	}
}

func (h *DHCPHandler) handleDiscover(packet dhcp4.Packet, options dhcp4.Options) dhcp4.Packet {
	mac := packet.CHAddr()

	// Check for static reservation
	ip, hasReservation := h.server.GetReservation(mac)

	if !hasReservation {
		// Allocate IP from pool
		ip = h.allocateIP(mac)
		if ip == nil {
			return nil
		}
	}

	// Build DHCP offer
	return h.buildOffer(packet, ip, options)
}

func (h *DHCPHandler) handleRequest(packet dhcp4.Packet, options dhcp4.Options) dhcp4.Packet {
	mac := packet.CHAddr()
	requestedIP := net.IP(options[dhcp4.OptionRequestedIPAddress])

	// Check for static reservation
	reservedIP, hasReservation := h.server.GetReservation(mac)

	var assignedIP net.IP
	if hasReservation {
		assignedIP = reservedIP
	} else if requestedIP != nil {
		assignedIP = requestedIP
	} else {
		assignedIP = h.allocateIP(mac)
	}

	if assignedIP == nil {
		return dhcp4.ReplyPacket(packet, dhcp4.NAK, h.server.getServerIP(), nil, 0, nil)
	}

	// Update lease
	h.server.mu.Lock()
	h.server.leases[mac.String()] = assignedIP
	h.server.mu.Unlock()

	// Build ACK
	return h.buildACK(packet, assignedIP, options)
}

func (h *DHCPHandler) handleRelease(packet dhcp4.Packet) {
	mac := packet.CHAddr()
	h.server.mu.Lock()
	delete(h.server.leases, mac.String())
	h.server.mu.Unlock()
}

func (h *DHCPHandler) allocateIP(mac net.HardwareAddr) net.IP {
	// Simple sequential allocation from range
	// Production implementation would be more sophisticated
	rangeStart := net.ParseIP(h.server.config.RangeStart)
	rangeEnd := net.ParseIP(h.server.config.RangeEnd)

	// Check existing lease
	h.server.mu.RLock()
	if existingIP, ok := h.server.leases[mac.String()]; ok {
		h.server.mu.RUnlock()
		return existingIP
	}
	h.server.mu.RUnlock()

	// Find next available IP
	for ip := rangeStart; !ip.Equal(rangeEnd); ip = nextIP(ip) {
		if !h.isIPInUse(ip) {
			return ip
		}
	}

	return nil
}

func (h *DHCPHandler) isIPInUse(ip net.IP) bool {
	h.server.mu.RLock()
	defer h.server.mu.RUnlock()

	for _, leased := range h.server.leases {
		if leased.Equal(ip) {
			return true
		}
	}

	for _, reserved := range h.server.reservations {
		if reserved.Equal(ip) {
			return true
		}
	}

	return false
}

func (h *DHCPHandler) buildOffer(packet dhcp4.Packet, ip net.IP, options dhcp4.Options) dhcp4.Packet {
	replyOptions := h.buildOptions(options)
	return dhcp4.ReplyPacket(packet, dhcp4.Offer, h.server.getServerIP(), ip, h.server.config.LeaseTime, replyOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList]))
}

func (h *DHCPHandler) buildACK(packet dhcp4.Packet, ip net.IP, options dhcp4.Options) dhcp4.Packet {
	replyOptions := h.buildOptions(options)
	return dhcp4.ReplyPacket(packet, dhcp4.ACK, h.server.getServerIP(), ip, h.server.config.LeaseTime, replyOptions.SelectOrderOrAll(options[dhcp4.OptionParameterRequestList]))
}

func (h *DHCPHandler) buildOptions(reqOptions dhcp4.Options) dhcp4.Options {
	options := dhcp4.Options{
		dhcp4.OptionSubnetMask:       []byte(net.ParseIP(h.server.config.Subnet).To4()),
		dhcp4.OptionRouter:           []byte(net.ParseIP(h.server.config.Router).To4()),
		dhcp4.OptionDomainNameServer: []byte(net.ParseIP(h.server.config.DNSServer).To4()),
	}

	// Add PXE boot options
	if h.server.config.TFTPServer != "" {
		options[dhcp4.OptionTFTPServerName] = []byte(h.server.config.TFTPServer)
	}

	// Check if UEFI or BIOS boot
	clientArch := reqOptions[93] // Client System Architecture Type
	if len(clientArch) > 0 && clientArch[0] == 0x00 && clientArch[1] == 0x07 {
		// UEFI x64
		options[dhcp4.OptionBootFileName] = []byte(h.server.config.UEFIBootFile)
	} else {
		// BIOS
		options[dhcp4.OptionBootFileName] = []byte(h.server.config.BIOSBootFile)
	}

	return options
}

func (s *DHCPServer) getServerIP() net.IP {
	return net.ParseIP(s.config.ServerIP).To4()
}

func nextIP(ip net.IP) net.IP {
	next := make(net.IP, len(ip))
	copy(next, ip)
	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}
	return next
}
