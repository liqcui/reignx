package agent

import (
	"fmt"
	"net"
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	agentpb "github.com/reignx/reignx/api/proto/gen"
)

// GetPrimaryIP returns the primary network interface IP address
func GetPrimaryIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, addr := range addrs {
		// Check if it's an IP address (not a network address)
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			// Check if it's IPv4
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no non-loopback IPv4 address found")
}

// GetOSInfo returns OS type and version information
func GetOSInfo() (osType, osVersion string, err error) {
	// OS type from runtime
	osType = runtime.GOOS

	// Get detailed OS info using gopsutil
	info, err := host.Info()
	if err != nil {
		return osType, "", fmt.Errorf("failed to get host info: %w", err)
	}

	// Combine platform and version for a descriptive OS version
	osVersion = fmt.Sprintf("%s %s", info.Platform, info.PlatformVersion)

	return osType, osVersion, nil
}

// MetricsSnapshot represents a snapshot of system metrics
type MetricsSnapshot struct {
	CPUPercent    float64
	MemoryPercent float64
	DiskPercent   float64
	NetworkRxMB   float64
	NetworkTxMB   float64
	UptimeSeconds int64
	RunningTasks  int32
}

// GetMetricsSnapshot collects current system metrics
func GetMetricsSnapshot() (*MetricsSnapshot, error) {
	snapshot := &MetricsSnapshot{}

	// CPU usage
	cpuPercents, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercents) > 0 {
		snapshot.CPUPercent = cpuPercents[0]
	}

	// Memory usage
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		snapshot.MemoryPercent = memInfo.UsedPercent
	}

	// Disk usage (root partition)
	diskInfo, err := disk.Usage("/")
	if err == nil {
		snapshot.DiskPercent = diskInfo.UsedPercent
	}

	// Uptime
	hostInfo, err := host.Info()
	if err == nil {
		snapshot.UptimeSeconds = int64(hostInfo.Uptime)
	}

	// Network stats would require tracking deltas over time
	// For now, we'll leave these at 0
	snapshot.NetworkRxMB = 0
	snapshot.NetworkTxMB = 0

	// Running tasks would be tracked by the agent task manager
	snapshot.RunningTasks = 0

	return snapshot, nil
}

// ToProtoMetricsSnapshot converts MetricsSnapshot to protobuf format
func (m *MetricsSnapshot) ToProto() *agentpb.MetricsSnapshot {
	return &agentpb.MetricsSnapshot{
		CpuPercent:    m.CPUPercent,
		MemoryPercent: m.MemoryPercent,
		DiskPercent:   m.DiskPercent,
		NetworkRxMb:   m.NetworkRxMB,
		NetworkTxMb:   m.NetworkTxMB,
		UptimeSeconds: m.UptimeSeconds,
		RunningTasks:  m.RunningTasks,
	}
}
