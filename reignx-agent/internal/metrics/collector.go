package metrics

import (
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"go.uber.org/zap"
)

// ServerMetrics contains system metrics
type ServerMetrics struct {
	CPUPercent    float64
	MemoryPercent float64
	MemoryUsedGB  float64
	MemoryTotalGB float64
	DiskPercent   float64
	DiskUsedGB    float64
	DiskTotalGB   float64
	NetworkRxMB   float64
	NetworkTxMB   float64
	Uptime        int64
	NumCPUs       int
	NumGoroutines int
	Timestamp     time.Time
}

// ProcessMetrics contains process-specific metrics
type ProcessMetrics struct {
	PID           int32
	CPUPercent    float64
	MemoryPercent float64
	MemoryUsedMB  float64
	NumThreads    int32
	NumFDs        int32
	Timestamp     time.Time
}

// Collector collects system and process metrics
type Collector struct {
	logger           *zap.Logger
	lastNetworkStats *net.IOCountersStat
	lastCollectTime  time.Time
}

// NewCollector creates a new metrics collector
func NewCollector(logger *zap.Logger) *Collector {
	return &Collector{
		logger:          logger,
		lastCollectTime: time.Now(),
	}
}

// Collect collects current system metrics
func (c *Collector) Collect() *ServerMetrics {
	metrics := &ServerMetrics{
		Timestamp:     time.Now(),
		NumGoroutines: runtime.NumGoroutine(),
	}

	// CPU metrics
	if cpuPercents, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercents) > 0 {
		metrics.CPUPercent = cpuPercents[0]
	} else if err != nil {
		c.logger.Warn("Failed to get CPU metrics", zap.Error(err))
	}

	// CPU count
	if cpuCount, err := cpu.Counts(true); err == nil {
		metrics.NumCPUs = cpuCount
	}

	// Memory metrics
	if vmStat, err := mem.VirtualMemory(); err == nil {
		metrics.MemoryPercent = vmStat.UsedPercent
		metrics.MemoryUsedGB = float64(vmStat.Used) / 1024 / 1024 / 1024
		metrics.MemoryTotalGB = float64(vmStat.Total) / 1024 / 1024 / 1024
	} else {
		c.logger.Warn("Failed to get memory metrics", zap.Error(err))
	}

	// Disk metrics (root partition)
	if diskStat, err := disk.Usage("/"); err == nil {
		metrics.DiskPercent = diskStat.UsedPercent
		metrics.DiskUsedGB = float64(diskStat.Used) / 1024 / 1024 / 1024
		metrics.DiskTotalGB = float64(diskStat.Total) / 1024 / 1024 / 1024
	} else {
		c.logger.Warn("Failed to get disk metrics", zap.Error(err))
	}

	// Network metrics
	if netStats, err := net.IOCounters(false); err == nil && len(netStats) > 0 {
		currentStats := &netStats[0]

		if c.lastNetworkStats != nil {
			// Calculate delta
			timeDelta := time.Since(c.lastCollectTime).Seconds()
			if timeDelta > 0 {
				rxDelta := float64(currentStats.BytesRecv - c.lastNetworkStats.BytesRecv)
				txDelta := float64(currentStats.BytesSent - c.lastNetworkStats.BytesSent)

				// Convert to MB/s
				metrics.NetworkRxMB = (rxDelta / 1024 / 1024) / timeDelta
				metrics.NetworkTxMB = (txDelta / 1024 / 1024) / timeDelta
			}
		}

		c.lastNetworkStats = currentStats
		c.lastCollectTime = time.Now()
	} else if err != nil {
		c.logger.Warn("Failed to get network metrics", zap.Error(err))
	}

	// Uptime
	if uptimeStat, err := host.Uptime(); err == nil {
		metrics.Uptime = int64(uptimeStat)
	} else {
		c.logger.Warn("Failed to get uptime", zap.Error(err))
	}

	return metrics
}

// CollectProcess collects current process metrics
func (c *Collector) CollectProcess() *ProcessMetrics {
	metrics := &ProcessMetrics{
		Timestamp: time.Now(),
	}

	// TODO: Collect process-specific metrics using gopsutil/process
	// For now, just return basic metrics

	return metrics
}

// GetSystemInfo returns static system information
func (c *Collector) GetSystemInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// Host information
	if hostInfo, err := host.Info(); err == nil {
		info["hostname"] = hostInfo.Hostname
		info["os"] = hostInfo.OS
		info["platform"] = hostInfo.Platform
		info["platform_family"] = hostInfo.PlatformFamily
		info["platform_version"] = hostInfo.PlatformVersion
		info["kernel_version"] = hostInfo.KernelVersion
		info["kernel_arch"] = hostInfo.KernelArch
	}

	// CPU information
	if cpuInfo, err := cpu.Info(); err == nil && len(cpuInfo) > 0 {
		info["cpu_model"] = cpuInfo[0].ModelName
		info["cpu_cores"] = cpuInfo[0].Cores
		info["cpu_mhz"] = cpuInfo[0].Mhz
	}

	// Memory information
	if vmStat, err := mem.VirtualMemory(); err == nil {
		info["memory_total_gb"] = float64(vmStat.Total) / 1024 / 1024 / 1024
	}

	// Disk information
	if diskStat, err := disk.Usage("/"); err == nil {
		info["disk_total_gb"] = float64(diskStat.Total) / 1024 / 1024 / 1024
	}

	return info
}

// GetHealthStatus returns the health status based on current metrics
func (c *Collector) GetHealthStatus() string {
	metrics := c.Collect()

	// Check if any critical thresholds are exceeded
	if metrics.CPUPercent > 90 {
		return "critical"
	}
	if metrics.MemoryPercent > 90 {
		return "critical"
	}
	if metrics.DiskPercent > 90 {
		return "critical"
	}

	// Check if any warning thresholds are exceeded
	if metrics.CPUPercent > 75 {
		return "warning"
	}
	if metrics.MemoryPercent > 75 {
		return "warning"
	}
	if metrics.DiskPercent > 80 {
		return "warning"
	}

	return "healthy"
}

// FormatMetrics formats metrics for human-readable output
func FormatMetrics(m *ServerMetrics) string {
	return fmt.Sprintf(
		"CPU: %.1f%% | Memory: %.1f%% (%.1f/%.1fGB) | Disk: %.1f%% (%.1f/%.1fGB) | Network: RX %.2f MB/s, TX %.2f MB/s | Uptime: %s",
		m.CPUPercent,
		m.MemoryPercent,
		m.MemoryUsedGB,
		m.MemoryTotalGB,
		m.DiskPercent,
		m.DiskUsedGB,
		m.DiskTotalGB,
		m.NetworkRxMB,
		m.NetworkTxMB,
		formatUptime(m.Uptime),
	)
}

// formatUptime formats uptime seconds into human-readable format
func formatUptime(seconds int64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
