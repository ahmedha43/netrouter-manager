package network

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

var defaultLeasePaths = []string{
	"/var/lib/misc/dnsmasq.leases",
	"/var/run/dnsmasq.leases",
	"/tmp/dnsmasq.leases",
	"/run/dnsmasq.leases",
}

func (m *Manager) ListDHCPLeases() ([]protocol.DHCPLease, error) {
	for _, path := range defaultLeasePaths {
		if _, err := os.Stat(path); err == nil {
			return parseDnsmasqLeases(path)
		}
	}
	return []protocol.DHCPLease{}, nil
}

func parseDnsmasqLeases(path string) ([]protocol.DHCPLease, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var leases []protocol.DHCPLease
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// dnsmasq.leases format:
		// <expiry-epoch> <mac> <ip> <hostname> <client-id>
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		expiry, _ := strconv.ParseInt(fields[0], 10, 64)
		mac := fields[1]
		ip := fields[2]
		hostname := fields[3]
		if hostname == "*" {
			hostname = "unknown"
		}

		leases = append(leases, protocol.DHCPLease{
			IPAddress:      ip,
			MACAddress:     mac,
			Hostname:       hostname,
			ExpirationTime: expiry,
			IsStatic:       expiry == 0,
		})
	}
	return leases, scanner.Err()
}
