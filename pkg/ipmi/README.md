# IPMI/BMC Integration

Out-of-band server management using IPMI (Intelligent Platform Management Interface) for hardware-level control of bare metal servers.

## Features

- **Power Management**: Power on/off, power cycle, hard reset
- **Boot Device Control**: Set boot device (PXE, disk, CDROM, BIOS)
- **Hardware Monitoring**: Sensor readings (temperature, voltage, fan speed)
- **System Event Log**: Retrieve hardware events and errors
- **FRU Information**: Field Replaceable Unit details (serial numbers, part numbers)

## Usage

### Basic Power Control

```go
package main

import (
    "context"
    "github.com/liqcui/bm-distributed-solution/pkg/ipmi"
)

func main() {
    // Create IPMI client
    client := ipmi.NewClient(&ipmi.Config{
        Host:     "192.168.1.100",
        Port:     623,
        Username: "admin",
        Password: "password",
        Timeout:  10 * time.Second,
    })

    ctx := context.Background()

    // Connect to BMC
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Get power status
    status, err := client.GetPowerStatus(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Power status: %s\n", status)

    // Power on the server
    if status == ipmi.PowerOff {
        if err := client.PowerOn(ctx); err != nil {
            log.Fatal(err)
        }
    }
}
```

### Boot Device Management

```go
// Set PXE boot for OS installation
if err := client.SetBootDevice(ctx, ipmi.BootDevicePXE, false); err != nil {
    log.Fatal(err)
}

// Power cycle to boot from PXE
if err := client.PowerCycle(ctx); err != nil {
    log.Fatal(err)
}

// After installation, set boot to disk (persistent)
if err := client.SetBootDevice(ctx, ipmi.BootDeviceDisk, true); err != nil {
    log.Fatal(err)
}
```

### Using IPMI Manager (Multiple Servers)

```go
manager := ipmi.NewManager()

// Add servers
manager.AddServer("server1", &ipmi.Config{
    Host:     "192.168.1.100",
    Username: "admin",
    Password: "password",
})

manager.AddServer("server2", &ipmi.Config{
    Host:     "192.168.1.101",
    Username: "admin",
    Password: "password",
})

// Power on all servers
for _, serverID := range []string{"server1", "server2"} {
    if err := manager.PowerOn(ctx, serverID); err != nil {
        log.Printf("Failed to power on %s: %v", serverID, err)
    }
}

// Trigger OS installation on server1
if err := manager.InstallOS(ctx, "server1"); err != nil {
    log.Fatal(err)
}

// Check server status
status, err := manager.GetServerStatus(ctx, "server1")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Server1 - Power: %s, Boot Device: %s\n",
    status.PowerState, status.BootDevice)
```

### Hardware Monitoring

```go
// Get sensor readings
sensors, err := client.GetSensorReadings(ctx)
if err != nil {
    log.Fatal(err)
}

for _, sensor := range sensors {
    fmt.Printf("%s: %.2f %s (%s)\n",
        sensor.Name, sensor.Value, sensor.Unit, sensor.Status)
}

// Get system event log
events, err := client.GetSystemEventLog(ctx)
if err != nil {
    log.Fatal(err)
}

for _, event := range events {
    fmt.Println(event)
}
```

## Integration with ReignX Platform

The IPMI integration is used by the ReignX platform for:

1. **Automated OS Installation**: Set PXE boot and power cycle servers
2. **Power Management**: Remotely control server power states
3. **Hardware Health Monitoring**: Track server hardware status
4. **Disaster Recovery**: Force power off/on for unresponsive servers

## Supported Hardware

- Dell iDRAC
- HP iLO
- Supermicro IPMI
- IBM IMM
- Any IPMI 2.0 compliant BMC

## Security Considerations

- **Credentials**: Store IPMI credentials securely (Kubernetes Secrets, Vault)
- **Network Isolation**: Keep BMC network isolated from production
- **TLS**: Use IPMI over LAN with encryption when available
- **Access Control**: Limit IPMI access to management nodes only

## Troubleshooting

### Connection Timeout

```bash
# Test IPMI connectivity
ipmitool -I lanplus -H 192.168.1.100 -U admin -P password chassis status

# Check network connectivity
ping 192.168.1.100
telnet 192.168.1.100 623
```

### Authentication Failed

- Verify username/password
- Check IPMI user permissions (may need administrator role)
- Ensure IPMI over LAN is enabled in BIOS

### Command Not Supported

- Some vendors have limited IPMI support
- Check BMC firmware version (upgrade if needed)
- Use vendor-specific tools as fallback (ipmitool, racadm, hponcfg)

## References

- [IPMI Specification](https://www.intel.com/content/www/us/en/products/docs/servers/ipmi/ipmi-second-gen-interface-spec-v2-rev1-1.html)
- [goipmi Library](https://github.com/vmware/goipmi)
- [ipmitool Documentation](https://github.com/ipmitool/ipmitool)
