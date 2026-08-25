//go:build !linux

package network

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

var startTime = time.Now()

func ReadSystemStatus() (protocol.SystemStatus, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "NetRouter-Core"
	}
	uptime := uint64(time.Since(startTime).Seconds())

	return protocol.SystemStatus{
		Identity:     hostname,
		Architecture: runtime.GOARCH,
		Kernel:       runtime.GOOS,
		Uptime:       uptime,
		MemoryTotal:  512 * 1024 * 1024,
		MemoryFree:   384 * 1024 * 1024,
		Load1:        0.05,
		DefaultRoute: fmt.Sprintf("ether1 via 192.168.88.1"),
	}, nil
}
