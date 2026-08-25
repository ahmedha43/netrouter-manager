package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"
)

const (
	DiscoveryPort = 8444
)

type NeighborDevice struct {
	Identity     string `json:"identity"`
	MAC          string `json:"mac"`
	IPv4         string `json:"ipv4"`
	Architecture string `json:"architecture"`
	Version      string `json:"version"`
	Uptime       uint64 `json:"uptime"`
	Port         int    `json:"port"`
}

type DiscoveryService struct {
	mu        sync.RWMutex
	neighbors map[string]NeighborDevice
}

func NewDiscoveryService() *DiscoveryService {
	return &DiscoveryService{
		neighbors: make(map[string]NeighborDevice),
	}
}

// StartDiscoveryServer listens for discovery queries and sends periodic broadcast beacons
func StartDiscoveryServer(ctx context.Context, identity, version string) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: DiscoveryPort})
	if err != nil {
		return
	}
	defer conn.Close()

	// 1. Responder Loop
	go func() {
		buf := make([]byte, 1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
				n, remoteAddr, err := conn.ReadFromUDP(buf)
				if err == nil && n > 0 {
					// Send beacon back to requester
					beacon := makeBeacon(identity, version)
					encoded, _ := json.Marshal(beacon)
					_, _ = conn.WriteToUDP(encoded, remoteAddr)
				}
			}
		}
	}()

	// 2. Periodic Broadcast Beacon Loop (every 3s)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	broadcastAddr := &net.UDPAddr{IP: net.IPv4bcast, Port: DiscoveryPort}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			beacon := makeBeacon(identity, version)
			encoded, _ := json.Marshal(beacon)
			_, _ = conn.WriteToUDP(encoded, broadcastAddr)
		}
	}
}

func makeBeacon(identity, version string) NeighborDevice {
	mac := "00:00:00:00:00:00"
	ip := "192.168.88.1"

	if ifaces, err := net.Interfaces(); err == nil {
		for _, ifc := range ifaces {
			if ifc.Flags&net.FlagLoopback == 0 && len(ifc.HardwareAddr) > 0 {
				mac = ifc.HardwareAddr.String()
				if addrs, err := ifc.Addrs(); err == nil {
					for _, addr := range addrs {
						if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
							ip = ipnet.IP.String()
							break
						}
					}
				}
				break
			}
		}
	}

	st, _ := ReadSystemStatus()

	return NeighborDevice{
		Identity:     identity,
		MAC:          mac,
		IPv4:         ip,
		Architecture: runtime.GOARCH,
		Version:      version,
		Uptime:       st.Uptime,
		Port:         8443,
	}
}

// ScanNeighbors broadcasts a discovery probe and collects neighbor replies
func ScanNeighbors(timeout time.Duration) ([]NeighborDevice, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: 0})
	if err != nil {
		return nil, fmt.Errorf("listen udp: %w", err)
	}
	defer conn.Close()

	broadcastAddr := &net.UDPAddr{IP: net.IPv4bcast, Port: DiscoveryPort}
	probe := []byte("NETROUTER_DISCOVERY_PING")
	_, _ = conn.WriteToUDP(probe, broadcastAddr)

	devices := make(map[string]NeighborDevice)
	deadline := time.Now().Add(timeout)
	buf := make([]byte, 2048)

	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, _, err := conn.ReadFromUDP(buf)
		if err == nil && n > 0 {
			var dev NeighborDevice
			if err := json.Unmarshal(buf[:n], &dev); err == nil && dev.MAC != "" {
				devices[dev.MAC] = dev
			}
		}
	}

	var result []NeighborDevice
	for _, dev := range devices {
		result = append(result, dev)
	}
	return result, nil
}
