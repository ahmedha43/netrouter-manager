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
	return protocol.SystemStatus{Identity: hostname, Architecture: runtime.GOARCH, Kernel: kernelRelease(), Uptime: uint64(info.Uptime), MemoryTotal: uint64(info.Totalram) * uint64(info.Unit), MemoryFree: uint64(info.Freeram) * uint64(info.Unit), Load1: float64(info.Loads[0]) / 65536.0}, nil
}

func kernelRelease() string {
	contents, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(contents))
}
