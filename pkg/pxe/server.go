package pxe

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

// Server represents a PXE boot server
type Server struct {
	config       *Config
	dhcpServer   *DHCPServer
	tftpServer   *TFTPServer
	httpServer   *HTTPServer
	templateMgr  *TemplateManager
}

// Config contains PXE server configuration
type Config struct {
	// Network settings
	Interface      string
	DHCPRangeStart string
	DHCPRangeEnd   string
	DHCPSubnet     string
	DHCPRouter     string
	DHCPDNSServer  string
	ServerIP       string

	// Directory paths
	TFTPRoot       string
	HTTPRoot       string
	TemplateDir    string

	// Server ports
	TFTPPort       int
	HTTPPort       int

	// Boot files
	BIOSBootFile   string
	UEFIBootFile   string

	// OS images
	ImagesDir      string
}

// DefaultConfig returns default PXE server configuration
func DefaultConfig() *Config {
	return &Config{
		Interface:      "eth0",
		DHCPRangeStart: "192.168.1.100",
		DHCPRangeEnd:   "192.168.1.200",
		DHCPSubnet:     "192.168.1.0/24",
		DHCPRouter:     "192.168.1.1",
		DHCPDNSServer:  "8.8.8.8",
		ServerIP:       "192.168.1.10",
		TFTPRoot:       "/var/lib/reignx/tftp",
		HTTPRoot:       "/var/lib/reignx/http",
		TemplateDir:    "/etc/reignx/templates",
		TFTPPort:       69,
		HTTPPort:       8888,
		BIOSBootFile:   "pxelinux.0",
		UEFIBootFile:   "bootx64.efi",
		ImagesDir:      "/var/lib/reignx/images",
	}
}

// NewServer creates a new PXE boot server
func NewServer(config *Config) (*Server, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create required directories
	dirs := []string{
		config.TFTPRoot,
		config.HTTPRoot,
		config.TemplateDir,
		config.ImagesDir,
		filepath.Join(config.TFTPRoot, "pxelinux.cfg"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	server := &Server{
		config:      config,
		templateMgr: NewTemplateManager(config.TemplateDir),
	}

	return server, nil
}

// Start starts all PXE server components
func (s *Server) Start(ctx context.Context) error {
	// Start TFTP server
	tftpServer, err := NewTFTPServer(s.config.TFTPRoot, s.config.TFTPPort)
	if err != nil {
		return fmt.Errorf("failed to create TFTP server: %w", err)
	}
	s.tftpServer = tftpServer

	if err := s.tftpServer.Start(); err != nil {
		return fmt.Errorf("failed to start TFTP server: %w", err)
	}

	// Start HTTP server for kickstart files and images
	httpServer, err := NewHTTPServer(s.config.HTTPRoot, s.config.HTTPPort)
	if err != nil {
		return fmt.Errorf("failed to create HTTP server: %w", err)
	}
	s.httpServer = httpServer

	go func() {
		if err := s.httpServer.Start(); err != nil {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	// Start DHCP server
	dhcpServer, err := NewDHCPServer(&DHCPConfig{
		Interface:      s.config.Interface,
		ServerIP:       s.config.ServerIP,
		RangeStart:     s.config.DHCPRangeStart,
		RangeEnd:       s.config.DHCPRangeEnd,
		Subnet:         s.config.DHCPSubnet,
		Router:         s.config.DHCPRouter,
		DNSServer:      s.config.DHCPDNSServer,
		TFTPServer:     s.config.ServerIP,
		BIOSBootFile:   s.config.BIOSBootFile,
		UEFIBootFile:   s.config.UEFIBootFile,
	})
	if err != nil {
		return fmt.Errorf("failed to create DHCP server: %w", err)
	}
	s.dhcpServer = dhcpServer

	go func() {
		if err := s.dhcpServer.Start(); err != nil {
			fmt.Printf("DHCP server error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops all PXE server components
func (s *Server) Stop(ctx context.Context) error {
	var errors []error

	if s.dhcpServer != nil {
		if err := s.dhcpServer.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("DHCP stop error: %w", err))
		}
	}

	if s.tftpServer != nil {
		if err := s.tftpServer.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("TFTP stop error: %w", err))
		}
	}

	if s.httpServer != nil {
		if err := s.httpServer.Stop(ctx); err != nil {
			errors = append(errors, fmt.Errorf("HTTP stop error: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping PXE server: %v", errors)
	}

	return nil
}

// ServerConfig represents configuration for a specific server
type ServerConfig struct {
	ServerID   string
	MACAddress string
	Hostname   string
	IPAddress  string
	OSType     string // ubuntu, centos, rhel, debian
	OSVersion  string
	RootPass   string
	SSHKeys    []string
	Partitions []Partition
	Packages   []string
}

// Partition represents a disk partition
type Partition struct {
	MountPoint string
	Size       string // e.g., "20G", "100%"
	FSType     string // ext4, xfs, swap
}

// ConfigureServer configures PXE boot for a specific server
func (s *Server) ConfigureServer(config *ServerConfig) error {
	// Generate PXE boot configuration
	if err := s.generatePXEConfig(config); err != nil {
		return fmt.Errorf("failed to generate PXE config: %w", err)
	}

	// Generate kickstart/preseed file
	if err := s.generateInstallConfig(config); err != nil {
		return fmt.Errorf("failed to generate install config: %w", err)
	}

	// Add DHCP reservation
	mac, err := net.ParseMAC(config.MACAddress)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %w", err)
	}

	ip := net.ParseIP(config.IPAddress)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", config.IPAddress)
	}

	s.dhcpServer.AddReservation(mac, ip, config.Hostname)

	return nil
}

// generatePXEConfig generates PXE boot configuration for a server
func (s *Server) generatePXEConfig(config *ServerConfig) error {
	// Convert MAC address to PXE config filename format
	// Example: 01-aa-bb-cc-dd-ee-ff
	macFile := "01-" + config.MACAddress

	configPath := filepath.Join(s.config.TFTPRoot, "pxelinux.cfg", macFile)

	// Determine kernel and initrd paths based on OS
	kernelPath, initrdPath := s.getOSPaths(config.OSType, config.OSVersion)

	// Generate PXE config from template
	data := map[string]interface{}{
		"Hostname":       config.Hostname,
		"KernelPath":     kernelPath,
		"InitrdPath":     initrdPath,
		"KickstartURL":   fmt.Sprintf("http://%s:%d/kickstart/%s.cfg", s.config.ServerIP, s.config.HTTPPort, config.ServerID),
		"OSType":         config.OSType,
		"OSVersion":      config.OSVersion,
	}

	return s.templateMgr.RenderToFile("pxe-config.tmpl", configPath, data)
}

// generateInstallConfig generates kickstart/preseed configuration
func (s *Server) generateInstallConfig(config *ServerConfig) error {
	configPath := filepath.Join(s.config.HTTPRoot, "kickstart", config.ServerID+".cfg")

	// Ensure kickstart directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	var templateName string
	switch config.OSType {
	case "ubuntu", "debian":
		templateName = "preseed.tmpl"
	case "centos", "rhel":
		templateName = "kickstart.tmpl"
	default:
		return fmt.Errorf("unsupported OS type: %s", config.OSType)
	}

	data := map[string]interface{}{
		"Hostname":     config.Hostname,
		"RootPassword": config.RootPass,
		"SSHKeys":      config.SSHKeys,
		"Partitions":   config.Partitions,
		"Packages":     config.Packages,
		"ServerIP":     s.config.ServerIP,
		"HTTPPort":     s.config.HTTPPort,
	}

	return s.templateMgr.RenderToFile(templateName, configPath, data)
}

// getOSPaths returns kernel and initrd paths for an OS
func (s *Server) getOSPaths(osType, version string) (kernel, initrd string) {
	basePath := filepath.Join("/images", osType, version)

	switch osType {
	case "ubuntu", "debian":
		return filepath.Join(basePath, "linux"), filepath.Join(basePath, "initrd.gz")
	case "centos", "rhel":
		return filepath.Join(basePath, "vmlinuz"), filepath.Join(basePath, "initrd.img")
	default:
		return "", ""
	}
}

// RemoveServerConfig removes PXE configuration for a server
func (s *Server) RemoveServerConfig(serverID, macAddress string) error {
	// Remove PXE config file
	macFile := "01-" + macAddress
	configPath := filepath.Join(s.config.TFTPRoot, "pxelinux.cfg", macFile)
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PXE config: %w", err)
	}

	// Remove kickstart file
	kickstartPath := filepath.Join(s.config.HTTPRoot, "kickstart", serverID+".cfg")
	if err := os.Remove(kickstartPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove kickstart config: %w", err)
	}

	// Remove DHCP reservation
	mac, _ := net.ParseMAC(macAddress)
	s.dhcpServer.RemoveReservation(mac)

	return nil
}

// UploadOSImage uploads an OS image to the PXE server
func (s *Server) UploadOSImage(osType, version, isoPath string) error {
	targetDir := filepath.Join(s.config.ImagesDir, osType, version)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create image directory: %w", err)
	}

	// Extract ISO to target directory
	// This would use a library to mount and extract ISO contents
	// For now, this is a placeholder

	return fmt.Errorf("ISO extraction not yet implemented")
}

// GetServerStatus returns the installation status of a server
func (s *Server) GetServerStatus(serverID string) (string, error) {
	// Check if server has completed installation by checking for completion marker
	markerPath := filepath.Join(s.config.HTTPRoot, "status", serverID+".done")

	if _, err := os.Stat(markerPath); err == nil {
		return "completed", nil
	} else if os.IsNotExist(err) {
		return "in_progress", nil
	} else {
		return "unknown", err
	}
}

// TemplateManager manages installation templates
type TemplateManager struct {
	templateDir string
	templates   map[string]*template.Template
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(templateDir string) *TemplateManager {
	return &TemplateManager{
		templateDir: templateDir,
		templates:   make(map[string]*template.Template),
	}
}

// RenderToFile renders a template to a file
func (tm *TemplateManager) RenderToFile(templateName, outputPath string, data interface{}) error {
	tmpl, err := tm.getTemplate(templateName)
	if err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// getTemplate loads or retrieves a cached template
func (tm *TemplateManager) getTemplate(name string) (*template.Template, error) {
	if tmpl, ok := tm.templates[name]; ok {
		return tmpl, nil
	}

	templatePath := filepath.Join(tm.templateDir, name)
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	tm.templates[name] = tmpl
	return tmpl, nil
}
