package network

import (
	"context"
	"testing"

	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

type recordedCommand struct {
	name string
	args []string
}
type fakeExecutor struct{ command recordedCommand }

func (f *fakeExecutor) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.command = recordedCommand{name: name, args: args}
	return nil, nil
}

func TestSetLinkStateRejectsInvalidInterfaceNameBeforeExecution(t *testing.T) {
	executor := &fakeExecutor{}
	err := NewManager(executor).SetLinkState(context.Background(), protocol.SetLinkStateParams{Name: "wan; rm -rf /", Up: true})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if executor.command.name != "" {
		t.Fatal("executor must not be called for invalid input")
	}
}

func TestAssignAddressRejectsInvalidCIDR(t *testing.T) {
	err := NewManager(&fakeExecutor{}).AssignAddress(context.Background(), protocol.AssignAddressParams{Name: "eth0", Address: "192.168.1.10"})
	if err == nil {
		t.Fatal("expected CIDR validation error")
	}
}
