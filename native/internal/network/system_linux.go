//go:build linux

package network

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

func ReadSystemStatus() (protocol.SystemStatus, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return protocol.SystemStatus{}, fmt.Errorf("read hostname: %w", err)
	}
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return protocol.SystemStatus{}, fmt.Errorf("read system info: %w", err)
	}

	defaultRoute := readDefaultRoute()

	return protocol.SystemStatus{
		Identity:     hostname,
		Architecture: runtime.GOARCH,
		Kernel:       kernelRelease(),
		Uptime:       uint64(info.Uptime),
		MemoryTotal:  uint64(info.Totalram) * uint64(info.Unit),
		MemoryFree:   uint64(info.Freeram) * uint64(info.Unit),
		Load1:        float64(info.Loads[0]) / 65536.0,
		DefaultRoute: defaultRoute,
	}, nil
}

func readDefaultRoute() string {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[1] == "00000000" {
			iface := fields[0]
			gwHex := fields[2]
			if len(gwHex) == 8 {
				var b0, b1, b2, b3 byte
				_, _ = fmt.Sscanf(gwHex, "%02x%02x%02x%02x", &b3, &b2, &b1, &b0)
				gwIP := fmt.Sprintf("%d.%d.%d.%d", b0, b1, b2, b3)
				return fmt.Sprintf("%s via %s", iface, gwIP)
			}
			return iface
		}
	}
	return ""
}

func kernelRelease() string {
	contents, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(contents))
}
