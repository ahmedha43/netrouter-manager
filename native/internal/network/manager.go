// Package network contains the Linux-specific NetRouter OS network boundary.
// It validates every mutable input before invoking a tightly scoped ip command.
package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
	"github.com/ahmedha43/netrouter-manager/native/internal/runner"
)

var (
	ErrPermission = errors.New("network mutations require root privileges")
	namePattern   = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,15}$`)
)

type Manager struct {
	executor     runner.Executor
	dnsmasqPath  string
	firewallPath string
	rootCheck    func() error
}

func NewManager(executor runner.Executor) *Manager {
	return NewManagerWithPaths(executor, "/etc/dnsmasq.d/netrouter.conf", "/etc/nftables.d/netrouter.nft")
}

func NewManagerWithPaths(executor runner.Executor, dnsmasqPath, firewallPath string) *Manager {
	return &Manager{executor: executor, dnsmasqPath: dnsmasqPath, firewallPath: firewallPath, rootCheck: requireRoot}
}

func (m *Manager) ListInterfaces() ([]protocol.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list Linux interfaces: %w", err)
	}
	result := make([]protocol.Interface, 0, len(interfaces))
	for _, iface := range interfaces {
		addresses, err := iface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("read addresses for %s: %w", iface.Name, err)
		}
		values := make([]string, 0, len(addresses))
		for _, address := range addresses {
			values = append(values, address.String())
		}
		result = append(result, protocol.Interface{Name: iface.Name, Index: iface.Index, MAC: iface.HardwareAddr.String(), MTU: iface.MTU, Up: iface.Flags&net.FlagUp != 0, Running: iface.Flags&net.FlagRunning != 0, Addresses: values})
	}
	return result, nil
}

func (m *Manager) SetLinkState(ctx context.Context, params protocol.SetLinkStateParams) error {
	if err := validateName(params.Name); err != nil {
		return err
	}
	if err := m.checkRoot(); err != nil {
		return err
	}
	state := "down"
	if params.Up {
		state = "up"
	}
	if output, err := m.executor.Run(ctx, "ip", "link", "set", "dev", params.Name, state); err != nil {
		return fmt.Errorf("set %s %s: %w: %s", params.Name, state, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (m *Manager) AssignAddress(ctx context.Context, params protocol.AssignAddressParams) error {
	if err := validateName(params.Name); err != nil {
		return err
	}
	if _, _, err := net.ParseCIDR(params.Address); err != nil {
		return fmt.Errorf("invalid CIDR address: %w", err)
	}
	if err := m.checkRoot(); err != nil {
		return err
	}
	if output, err := m.executor.Run(ctx, "ip", "address", "replace", params.Address, "dev", params.Name); err != nil {
		return fmt.Errorf("assign address to %s: %w: %s", params.Name, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (m *Manager) ReplaceDefaultRoute(ctx context.Context, params protocol.ReplaceDefaultRouteParams) error {
	if err := validateName(params.Device); err != nil {
		return err
	}
	if ip := net.ParseIP(params.Gateway); ip == nil {
		return fmt.Errorf("invalid default gateway")
	}
	if err := m.checkRoot(); err != nil {
		return err
	}
	if output, err := m.executor.Run(ctx, "ip", "route", "replace", "default", "via", params.Gateway, "dev", params.Device); err != nil {
		return fmt.Errorf("replace default route: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func validateName(name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("invalid interface name")
	}
	return nil
}

func (m *Manager) checkRoot() error {
	if m.rootCheck == nil {
		return requireRoot()
	}
	return m.rootCheck()
}

func requireRoot() error {
	if os.Geteuid() != 0 {
		return ErrPermission
	}
	return nil
}
