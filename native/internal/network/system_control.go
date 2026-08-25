package network

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

func (m *Manager) SetIdentity(ctx context.Context, params protocol.SetIdentityParams) error {
	identity := strings.TrimSpace(params.Identity)
	if err := validateName(identity); err != nil {
		return fmt.Errorf("invalid router identity: %w", err)
	}
	if err := m.checkRoot(); err != nil {
		return err
	}

	// Update /etc/hostname
	if err := os.WriteFile("/etc/hostname", []byte(identity+"\n"), 0o644); err != nil {
		// Log or proceed
	}

	// Set live hostname
	if output, err := m.executor.Run(ctx, "hostname", identity); err != nil {
		return fmt.Errorf("set hostname: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (m *Manager) RebootSystem(ctx context.Context, params protocol.RebootParams) error {
	if err := m.checkRoot(); err != nil {
		return err
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		if params.Force {
			_, _ = m.executor.Run(context.Background(), "reboot", "-f")
		} else {
			_, _ = m.executor.Run(context.Background(), "reboot")
		}
	}()

	return nil
}
