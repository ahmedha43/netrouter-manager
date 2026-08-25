package network

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ahmedha43/netrouter-manager/native/internal/config"
)

type PPPoEManager struct {
	m *Manager
}

func (m *Manager) ApplyPPPoE(ctx context.Context, cfg config.PPPoEConfig, iface string) error {
	if err := validateName(iface); err != nil {
		return err
	}
	if cfg.Username == "" {
		return fmt.Errorf("pppoe username is required")
	}
	if cfg.MTU <= 0 {
		cfg.MTU = 1492
	}
	if err := m.checkRoot(); err != nil {
		return err
	}

	// 1. Write /etc/ppp/peers/netrouter-pppoe
	peersDir := "/etc/ppp/peers"
	if err := os.MkdirAll(peersDir, 0o750); err != nil {
		return fmt.Errorf("create ppp peers dir: %w", err)
	}

	peerContent := fmt.Sprintf(`# Managed by NetRouter OS
plugin rp-pppoe.so %s
user "%s"
noauth
defaultroute
replacedefaultroute
persist
maxfail 0
holdoff 5
mtu %d
mru %d
usepeerdns
`, iface, cfg.Username, cfg.MTU, cfg.MTU)

	peerFile := filepath.Join(peersDir, "netrouter-pppoe")
	if err := atomicWrite(peerFile, []byte(peerContent), 0o600); err != nil {
		return fmt.Errorf("write pppoe peer config: %w", err)
	}

	// 2. Write pap-secrets & chap-secrets
	secretEntry := fmt.Sprintf("\"%s\" * \"%s\"\n", cfg.Username, cfg.Password)
	_ = atomicWrite("/etc/ppp/pap-secrets", []byte(secretEntry), 0o600)
	_ = atomicWrite("/etc/ppp/chap-secrets", []byte(secretEntry), 0o600)

	// 3. Restart pppd
	_, _ = m.executor.Run(ctx, "killall", "pppd")
	if output, err := m.executor.Run(ctx, "pppd", "call", "netrouter-pppoe"); err != nil {
		return fmt.Errorf("start pppd: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (m *Manager) StopPPPoE(ctx context.Context) error {
	if err := m.checkRoot(); err != nil {
		return err
	}
	_, err := m.executor.Run(ctx, "killall", "pppd")
	return err
}
