package network

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

type serviceExecutor struct {
	commands []recordedCommand
	run      func(name string, args ...string) ([]byte, error)
}

func (f *serviceExecutor) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.commands = append(f.commands, recordedCommand{name: name, args: args})
	if f.run == nil {
		return nil, nil
	}
	return f.run(name, args...)
}

func testManagerWithPaths(executor *serviceExecutor, dnsmasqPath, firewallPath string) *Manager {
	manager := NewManagerWithPaths(executor, dnsmasqPath, firewallPath)
	manager.rootCheck = func() error { return nil }
	return manager
}

func TestRenderDHCPDNSProducesBoundInterfaceConfiguration(t *testing.T) {
	contents, err := renderDHCPDNS(protocol.DHCPDNSParams{Interface: "lan0", SubnetCIDR: "192.168.88.0/24", Gateway: "192.168.88.1", PoolStart: "192.168.88.100", PoolEnd: "192.168.88.180", LeaseTime: "12h", DNSServers: []string{"1.1.1.1", "9.9.9.9"}})
	if err != nil {
		t.Fatalf("render DHCP/DNS: %v", err)
	}
	for _, expected := range []string{"interface=lan0", "dhcp-range=192.168.88.100,192.168.88.180,12h", "dhcp-option=option:router,192.168.88.1", "dhcp-option=option:dns-server,1.1.1.1,9.9.9.9"} {
		if !strings.Contains(contents, expected) {
			t.Fatalf("missing %q in %q", expected, contents)
		}
	}
}

func TestRenderDHCPDNSRejectsOutOfSubnetPool(t *testing.T) {
	_, err := renderDHCPDNS(protocol.DHCPDNSParams{Interface: "lan0", SubnetCIDR: "192.168.88.0/24", Gateway: "192.168.88.1", PoolStart: "10.0.0.2", PoolEnd: "10.0.0.3", LeaseTime: "12h", DNSServers: []string{"1.1.1.1"}})
	if err == nil {
		t.Fatal("expected out-of-subnet DHCP pool error")
	}
}

func TestRenderFirewallRejectsUnsafeOrIncompleteInput(t *testing.T) {
	_, err := renderFirewall(protocol.FirewallParams{LANInterface: "lan0; drop", WANInterface: "wan0", ManagementTCPPort: 8443})
	if err == nil {
		t.Fatal("expected interface validation error")
	}
	_, err = renderFirewall(protocol.FirewallParams{LANInterface: "lan0", WANInterface: "wan0", ManagementTCPPort: 0})
	if err == nil {
		t.Fatal("expected missing management port error")
	}
}

func TestApplyDHCPDNSValidatesStagedFileBeforeReplacingActiveConfig(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "netrouter.conf")
	if err := os.WriteFile(path, []byte("known-good\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	executor := &serviceExecutor{run: func(name string, args ...string) ([]byte, error) {
		if name != "dnsmasq" {
			t.Fatalf("unexpected command: %s", name)
		}
		stagedPath := strings.TrimPrefix(args[len(args)-1], "--conf-file=")
		if stagedPath == path {
			t.Fatal("validation must use a staged file, not the active configuration")
		}
		contents, err := os.ReadFile(stagedPath)
		if err != nil || !strings.Contains(string(contents), "interface=lan0") {
			t.Fatalf("staged DHCP/DNS configuration was not available to validation: %v", err)
		}
		return []byte("invalid config"), errors.New("validation failed")
	}}
	err := testManagerWithPaths(executor, path, filepath.Join(directory, "netrouter.nft")).ApplyDHCPDNS(context.Background(), protocol.DHCPDNSParams{Interface: "lan0", SubnetCIDR: "192.168.88.0/24", Gateway: "192.168.88.1", PoolStart: "192.168.88.100", PoolEnd: "192.168.88.180", LeaseTime: "12h", DNSServers: []string{"1.1.1.1"}})
	if err == nil {
		t.Fatal("expected validation failure")
	}
	contents, readErr := os.ReadFile(path)
	if readErr != nil || string(contents) != "known-good\n" {
		t.Fatalf("active configuration changed after validation failure: %q, %v", contents, readErr)
	}
}

func TestApplyFirewallValidatesStagedFileBeforeReplacingActiveConfig(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "netrouter.nft")
	if err := os.WriteFile(path, []byte("known-good\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	executor := &serviceExecutor{run: func(name string, args ...string) ([]byte, error) {
		if name != "nft" || len(args) != 3 || args[0] != "-c" || args[1] != "-f" {
			t.Fatalf("unexpected validation command: %s %v", name, args)
		}
		if args[2] == path {
			t.Fatal("validation must use a staged file, not the active firewall configuration")
		}
		contents, err := os.ReadFile(args[2])
		if err != nil || !strings.Contains(string(contents), "table inet netrouter") {
			t.Fatalf("staged firewall configuration was not available to validation: %v", err)
		}
		return []byte("parse error"), errors.New("validation failed")
	}}
	err := testManagerWithPaths(executor, filepath.Join(directory, "netrouter.conf"), path).ApplyFirewall(context.Background(), protocol.FirewallParams{LANInterface: "lan0", WANInterface: "wan0", ManagementTCPPort: 8443})
	if err == nil {
		t.Fatal("expected validation failure")
	}
	contents, readErr := os.ReadFile(path)
	if readErr != nil || string(contents) != "known-good\n" {
		t.Fatalf("active firewall configuration changed after validation failure: %q, %v", contents, readErr)
	}
}
