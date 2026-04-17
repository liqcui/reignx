# PXE Boot System

Network-based OS installation system for bare metal servers using PXE (Preboot Execution Environment).

## Features

- **DHCP Server**: Dynamic IP assignment with PXE boot options
- **TFTP Server**: Boot file distribution (kernels, initrd)
- **HTTP Server**: Kickstart/Preseed files and OS images
- **Template Engine**: Automated installation configuration generation
- **Multi-OS Support**: Ubuntu, Debian, CentOS, RHEL
- **BIOS/UEFI Support**: Automatic boot file selection

## Architecture

```
┌─────────────┐
│ Bare Metal  │
│   Server    │
└──────┬──────┘
       │ 1. DHCP Request
       ▼
┌─────────────┐
│ DHCP Server │ ─── PXE Boot Options (TFTP Server, Boot File)
└──────┬──────┘
       │ 2. TFTP Boot File Request
       ▼
┌─────────────┐
│ TFTP Server │ ─── pxelinux.0, kernel, initrd
└──────┬──────┘
       │ 3. Kickstart URL from Boot Config
       ▼
┌─────────────┐
│ HTTP Server │ ─── Kickstart/Preseed, OS Images
└─────────────┘
       │ 4. OS Installation
       ▼
  [Install Complete]
```

## Usage

### Start PXE Server

```go
package main

import (
    "context"
    "github.com/liqcui/bm-distributed-solution/pkg/pxe"
)

func main() {
    // Create PXE server with configuration
    config := &pxe.Config{
        Interface:      "eth0",
        ServerIP:       "192.168.1.10",
        DHCPRangeStart: "192.168.1.100",
        DHCPRangeEnd:   "192.168.1.200",
        DHCPSubnet:     "192.168.1.0/24",
        DHCPRouter:     "192.168.1.1",
        TFTPRoot:       "/var/lib/reignx/tftp",
        HTTPRoot:       "/var/lib/reignx/http",
    }

    server, err := pxe.NewServer(config)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Start all services (DHCP, TFTP, HTTP)
    if err := server.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // Wait for shutdown signal
    <-ctx.Done()
    server.Stop(ctx)
}
```

### Configure Server for Installation

```go
// Configure PXE boot for a specific server
serverConfig := &pxe.ServerConfig{
    ServerID:   "server-001",
    MACAddress: "aa:bb:cc:dd:ee:ff",
    Hostname:   "web-server-01",
    IPAddress:  "192.168.1.150",
    OSType:     "ubuntu",
    OSVersion:  "22.04",
    RootPass:   "$6$encrypted$password",
    SSHKeys: []string{
        "ssh-rsa AAAAB3... admin@example.com",
    },
    Partitions: []pxe.Partition{
        {MountPoint: "/boot", Size: "1G", FSType: "ext4"},
        {MountPoint: "swap", Size: "4G", FSType: "swap"},
        {MountPoint: "/", Size: "100G", FSType: "ext4"},
    },
    Packages: []string{
        "openssh-server",
        "curl",
        "vim",
    },
}

if err := server.ConfigureServer(serverConfig); err != nil {
    log.Fatal(err)
}

// Trigger installation via IPMI
// (Server will PXE boot and auto-install)
```

### Monitor Installation Progress

```go
// Check installation status
status, err := server.GetServerStatus("server-001")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Installation status: %s\n", status)
// Output: "in_progress" or "completed"
```

### Remove Server Configuration

```go
// After installation completes
if err := server.RemoveServerConfig("server-001", "aa:bb:cc:dd:ee:ff"); err != nil {
    log.Fatal(err)
}
```

## Installation Flow

1. **DHCP Discovery**: Server sends DHCP discover packet
2. **DHCP Offer**: PXE server responds with IP and boot options
3. **TFTP Boot**: Server downloads boot files via TFTP
4. **Boot Config**: PXE boot config points to kickstart URL
5. **Kickstart**: Server downloads kickstart/preseed from HTTP
6. **OS Installation**: Automated installation based on kickstart
7. **Post-Install**: Agent installation, SSH key setup
8. **Completion**: Server signals completion via HTTP callback
9. **Reboot**: Server reboots into installed OS

## Supported Operating Systems

### Ubuntu/Debian (Preseed)

```go
ServerConfig{
    OSType:    "ubuntu",
    OSVersion: "22.04",
    // ... other config
}
```

**Template**: `preseed.tmpl`

### CentOS/RHEL (Kickstart)

```go
ServerConfig{
    OSType:    "centos",
    OSVersion: "7",
    // ... other config
}
```

**Template**: `kickstart.tmpl`

## Directory Structure

```
/var/lib/reignx/
├── tftp/                    # TFTP root
│   ├── pxelinux.0          # BIOS boot loader
│   ├── bootx64.efi         # UEFI boot loader
│   ├── pxelinux.cfg/       # PXE boot configs
│   │   └── 01-aa-bb-cc-dd-ee-ff  # MAC-specific config
│   └── images/             # OS kernels and initrd
│       ├── ubuntu/
│       │   └── 22.04/
│       │       ├── linux
│       │       └── initrd.gz
│       └── centos/
│           └── 7/
│               ├── vmlinuz
│               └── initrd.img
├── http/                   # HTTP root
│   ├── kickstart/          # Kickstart files
│   │   └── server-001.cfg
│   ├── images/             # OS installation files
│   │   ├── ubuntu/22.04/
│   │   └── centos/7/
│   └── status/             # Installation completion markers
│       └── server-001.done
└── templates/              # Installation templates
    ├── kickstart.tmpl
    ├── preseed.tmpl
    └── pxe-config.tmpl
```

## Integration with IPMI

```go
// Complete bare metal provisioning workflow
import (
    "github.com/liqcui/bm-distributed-solution/pkg/pxe"
    "github.com/liqcui/bm-distributed-solution/pkg/ipmi"
)

// 1. Configure PXE boot
server.ConfigureServer(serverConfig)

// 2. Set boot device to PXE via IPMI
ipmiClient := ipmi.NewClient(&ipmi.Config{
    Host:     "192.168.1.200",
    Username: "admin",
    Password: "password",
})
ipmiClient.Connect(ctx)
ipmiClient.SetBootDevice(ctx, ipmi.BootDevicePXE, false)

// 3. Power cycle server
ipmiClient.PowerCycle(ctx)

// 4. Monitor installation
for {
    status, _ := server.GetServerStatus("server-001")
    if status == "completed" {
        break
    }
    time.Sleep(30 * time.Second)
}

// 5. Set boot to disk for normal operation
ipmiClient.SetBootDevice(ctx, ipmi.BootDeviceDisk, true)
```

## Network Requirements

- **DHCP**: Port 67 (UDP)
- **TFTP**: Port 69 (UDP)
- **HTTP**: Port 8888 (TCP, configurable)
- **Network**: Layer 2 connectivity to managed servers
- **VLAN**: Dedicated management VLAN recommended

## Security Considerations

- **Network Isolation**: Keep PXE network isolated
- **TFTP Read-Only**: Write operations disabled for security
- **HTTP Access Control**: Restrict access to management network
- **Credentials**: Don't store passwords in plain text
- **Post-Install**: Change default passwords after installation

## Troubleshooting

### Server Not PXE Booting

```bash
# Check DHCP server
tcpdump -i eth0 port 67 and port 68

# Check TFTP server
tftp 192.168.1.10 -c get pxelinux.0

# Check boot order in BIOS
# Ensure PXE/Network Boot is enabled
```

### Installation Hangs

- Check HTTP server accessibility
- Verify kickstart/preseed URL is correct
- Check firewall rules
- Review installation logs in `/var/log/installer/`

### Wrong Boot File

- BIOS vs UEFI detection issue
- Manually specify boot file in DHCP config
- Check client architecture type in DHCP packet

## Performance

- **Concurrent Installations**: 50+ servers simultaneously
- **Network Bandwidth**: ~100MB per installation (varies by OS)
- **Installation Time**:
  - Ubuntu: 10-15 minutes
  - CentOS: 15-20 minutes
  - RHEL: 15-20 minutes

## References

- [PXE Specification](https://www.intel.com/content/dam/www/public/us/en/documents/product-briefs/preboot-execution-environment-pxe-server.pdf)
- [Kickstart Documentation](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/7/html/installation_guide/chap-kickstart-installations)
- [Preseed Documentation](https://wiki.debian.org/DebianInstaller/Preseed)
- [SYSLINUX](https://wiki.syslinux.org/wiki/index.php?title=PXELINUX)
