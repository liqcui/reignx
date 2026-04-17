package agent

import (
	"testing"
)

func TestGetPrimaryIP(t *testing.T) {
	ip, err := GetPrimaryIP()
	if err != nil {
		t.Skipf("Skipping GetPrimaryIP test: %v", err)
		return
	}

	if ip == "" {
		t.Error("Expected non-empty IP address")
	}

	// Basic validation - should look like an IP
	if len(ip) < 7 {
		t.Errorf("IP address seems invalid: %s", ip)
	}

	t.Logf("Primary IP: %s", ip)
}

func TestGetOSInfo(t *testing.T) {
	osType, osVersion, err := GetOSInfo()
	if err != nil {
		t.Fatalf("GetOSInfo failed: %v", err)
	}

	if osType == "" {
		t.Error("Expected non-empty OS type")
	}

	if osVersion == "" {
		t.Error("Expected non-empty OS version")
	}

	t.Logf("OS Type: %s, Version: %s", osType, osVersion)

	// Verify OS type is one of the expected values
	validOSTypes := map[string]bool{
		"linux":   true,
		"darwin":  true,
		"windows": true,
	}

	if !validOSTypes[osType] {
		t.Errorf("Unexpected OS type: %s", osType)
	}
}

func TestGetMetricsSnapshot(t *testing.T) {
	snapshot, err := GetMetricsSnapshot()
	if err != nil {
		t.Fatalf("GetMetricsSnapshot failed: %v", err)
	}

	if snapshot == nil {
		t.Fatal("Expected non-nil snapshot")
	}

	// CPU percent should be between 0 and 100
	if snapshot.CPUPercent < 0 || snapshot.CPUPercent > 100 {
		t.Errorf("CPU percent out of range: %f", snapshot.CPUPercent)
	}

	// Memory percent should be between 0 and 100
	if snapshot.MemoryPercent < 0 || snapshot.MemoryPercent > 100 {
		t.Errorf("Memory percent out of range: %f", snapshot.MemoryPercent)
	}

	// Disk percent should be between 0 and 100
	if snapshot.DiskPercent < 0 || snapshot.DiskPercent > 100 {
		t.Errorf("Disk percent out of range: %f", snapshot.DiskPercent)
	}

	// Uptime should be positive
	if snapshot.UptimeSeconds < 0 {
		t.Errorf("Uptime should be positive: %d", snapshot.UptimeSeconds)
	}

	t.Logf("Metrics - CPU: %.2f%%, Memory: %.2f%%, Disk: %.2f%%, Uptime: %ds",
		snapshot.CPUPercent, snapshot.MemoryPercent, snapshot.DiskPercent, snapshot.UptimeSeconds)
}

func TestMetricsSnapshot_ToProto(t *testing.T) {
	snapshot := &MetricsSnapshot{
		CPUPercent:    45.5,
		MemoryPercent: 60.2,
		DiskPercent:   70.3,
		NetworkRxMB:   100.5,
		NetworkTxMB:   50.2,
		UptimeSeconds: 3600,
		RunningTasks:  5,
	}

	proto := snapshot.ToProto()

	if proto.CpuPercent != 45.5 {
		t.Errorf("Expected CPU 45.5, got %f", proto.CpuPercent)
	}
	if proto.MemoryPercent != 60.2 {
		t.Errorf("Expected Memory 60.2, got %f", proto.MemoryPercent)
	}
	if proto.DiskPercent != 70.3 {
		t.Errorf("Expected Disk 70.3, got %f", proto.DiskPercent)
	}
	if proto.UptimeSeconds != 3600 {
		t.Errorf("Expected Uptime 3600, got %d", proto.UptimeSeconds)
	}
	if proto.RunningTasks != 5 {
		t.Errorf("Expected RunningTasks 5, got %d", proto.RunningTasks)
	}
}
