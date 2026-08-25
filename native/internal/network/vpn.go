package network

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ahmedha43/netrouter-manager/native/internal/config"
)

func (m *Manager) ApplyWireGuard(ctx context.Context, cfg config.WireGuardConfig) error {
	if !cfg.Enabled {
		_, _ = m.executor.Run(ctx, "ip", "link", "del", "dev", cfg.Interface)
		return nil
	}

	if err := validateName(cfg.Interface); err != nil {
		return err
	}
	if err := m.checkRoot(); err != nil {
		return err
	}

	wgDir := "/etc/wireguard"
	if err := os.MkdirAll(wgDir, 0o700); err != nil {
		return fmt.Errorf("create wireguard dir: %w", err)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("[Interface]\nPrivateKey = %s\nListenPort = %d\n", cfg.PrivateKey, cfg.ListenPort))
	for _, peer := range cfg.Peers {
		b.WriteString(fmt.Sprintf("\n[Peer]\nPublicKey = %s\nAllowedIPs = %s\n", peer.PublicKey, strings.Join(peer.AllowedIPs, ", ")))
		if peer.Endpoint != "" {
			b.WriteString(fmt.Sprintf("Endpoint = %s\n", peer.Endpoint))
		}
	}

	confPath := filepath.Join(wgDir, cfg.Interface+".conf")
	if err := atomicWrite(confPath, []byte(b.String()), 0o600); err != nil {
		return fmt.Errorf("write wireguard conf: %w", err)
	}

	// Ensure wg interface exists
	_, _ = m.executor.Run(ctx, "ip", "link", "add", "dev", cfg.Interface, "type", "wireguard")
	if output, err := m.executor.Run(ctx, "wg", "setconf", cfg.Interface, confPath); err != nil {
		return fmt.Errorf("apply wg config: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if cfg.Address != "" {
		_, _ = m.executor.Run(ctx, "ip", "address", "replace", cfg.Address, "dev", cfg.Interface)
	}
	_, _ = m.executor.Run(ctx, "ip", "link", "set", "dev", cfg.Interface, "up")

	return nil
}
