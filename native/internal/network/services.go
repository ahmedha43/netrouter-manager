package network

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

func (m *Manager) ApplyDHCPDNS(ctx context.Context, params protocol.DHCPDNSParams) error {
	contents, err := renderDHCPDNS(params)
	if err != nil {
		return err
	}
	if err := m.checkRoot(); err != nil {
		return err
	}
	stagedPath, cleanup, err := stageWrite(m.dnsmasqPath, []byte(contents), 0o640)
	if err != nil {
		return fmt.Errorf("stage dnsmasq configuration: %w", err)
	}
	defer cleanup()
	if output, err := m.executor.Run(ctx, "dnsmasq", "--test", "--conf-file="+stagedPath); err != nil {
		return fmt.Errorf("validate dnsmasq configuration: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if err := commitStaged(stagedPath, m.dnsmasqPath); err != nil {
		return fmt.Errorf("commit dnsmasq configuration: %w", err)
	}
	if output, err := m.executor.Run(ctx, "killall", "-HUP", "dnsmasq"); err != nil {
		return fmt.Errorf("reload dnsmasq: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (m *Manager) ApplyFirewall(ctx context.Context, params protocol.FirewallParams) error {
	contents, err := renderFirewall(params)
	if err != nil {
		return err
	}
	if err := m.checkRoot(); err != nil {
		return err
	}
	stagedPath, cleanup, err := stageWrite(m.firewallPath, []byte(contents), 0o640)
	if err != nil {
		return fmt.Errorf("stage nftables configuration: %w", err)
	}
	defer cleanup()
	if output, err := m.executor.Run(ctx, "nft", "-c", "-f", stagedPath); err != nil {
		return fmt.Errorf("validate nftables configuration: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if output, err := m.executor.Run(ctx, "nft", "-f", stagedPath); err != nil {
		return fmt.Errorf("apply nftables configuration: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if err := commitStaged(stagedPath, m.firewallPath); err != nil {
		return fmt.Errorf("commit nftables configuration: %w", err)
	}
	return nil
}

func renderDHCPDNS(params protocol.DHCPDNSParams) (string, error) {
	if err := validateName(params.Interface); err != nil {
		return "", err
	}
	prefix, err := netip.ParsePrefix(params.SubnetCIDR)
	if err != nil || !prefix.Addr().Is4() {
		return "", fmt.Errorf("invalid IPv4 subnet")
	}
	gateway, err := netip.ParseAddr(params.Gateway)
	if err != nil || !gateway.Is4() || !prefix.Contains(gateway) {
		return "", fmt.Errorf("gateway must be inside the subnet")
	}
	start, err := netip.ParseAddr(params.PoolStart)
	if err != nil || !start.Is4() || !prefix.Contains(start) {
		return "", fmt.Errorf("pool start must be inside the subnet")
	}
	end, err := netip.ParseAddr(params.PoolEnd)
	if err != nil || !end.Is4() || !prefix.Contains(end) || start.Compare(end) > 0 {
		return "", fmt.Errorf("invalid DHCP pool range")
	}
	if params.LeaseTime == "" || strings.ContainsAny(params.LeaseTime, "\n\r") {
		return "", fmt.Errorf("invalid lease time")
	}
	dns := make([]string, 0, len(params.DNSServers))
	for _, value := range params.DNSServers {
		address, err := netip.ParseAddr(value)
		if err != nil || !address.Is4() {
			return "", fmt.Errorf("invalid DNS server")
		}
		dns = append(dns, address.String())
	}
	if len(dns) == 0 {
		return "", fmt.Errorf("at least one DNS server is required")
	}
	return fmt.Sprintf("# Managed by NetRouter OS; edits are replaced on apply.\ninterface=%s\nbind-interfaces\ndhcp-range=%s,%s,%s\ndhcp-option=option:router,%s\ndhcp-option=option:dns-server,%s\n", params.Interface, start, end, params.LeaseTime, gateway, strings.Join(dns, ",")), nil
}

func renderFirewall(params protocol.FirewallParams) (string, error) {
	if err := validateName(params.LANInterface); err != nil {
		return "", err
	}
	if err := validateName(params.WANInterface); err != nil {
		return "", err
	}
	if params.LANInterface == params.WANInterface {
		return "", fmt.Errorf("LAN and WAN interfaces must differ")
	}
	if params.ManagementTCPPort == 0 {
		return "", fmt.Errorf("management TCP port is required")
	}
	return fmt.Sprintf("table inet netrouter {\n  chain input {\n    type filter hook input priority filter; policy drop;\n    iifname \"lo\" accept\n    ct state established,related accept\n    iifname \"%s\" tcp dport %d accept\n    ip protocol icmp accept\n  }\n  chain forward {\n    type filter hook forward priority filter; policy drop;\n    ct state established,related accept\n    iifname \"%s\" oifname \"%s\" accept\n  }\n  chain postrouting {\n    type nat hook postrouting priority srcnat;\n    oifname \"%s\" masquerade\n  }\n}\n", params.LANInterface, params.ManagementTCPPort, params.LANInterface, params.WANInterface, params.WANInterface), nil
}

func atomicWrite(path string, contents []byte, mode os.FileMode) error {
	temporaryName, cleanup, err := stageWrite(path, contents, mode)
	if err != nil {
		return err
	}
	defer cleanup()
	return commitStaged(temporaryName, path)
}

func stageWrite(path string, contents []byte, mode os.FileMode) (string, func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", nil, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".netrouter-*")
	if err != nil {
		return "", nil, err
	}
	temporaryName := temporary.Name()
	cleanup := func() { _ = os.Remove(temporaryName) }
	if _, err := temporary.Write(contents); err != nil {
		_ = temporary.Close()
		cleanup()
		return "", nil, err
	}
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		cleanup()
		return "", nil, err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		cleanup()
		return "", nil, err
	}
	if err := temporary.Close(); err != nil {
		cleanup()
		return "", nil, err
	}
	return temporaryName, cleanup, nil
}

func commitStaged(stagedPath, destinationPath string) error {
	if err := os.Rename(stagedPath, destinationPath); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(destinationPath))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
