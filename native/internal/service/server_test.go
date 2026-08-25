package service

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ahmedha43/netrouter-manager/native/internal/network"
	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
	"github.com/ahmedha43/netrouter-manager/native/internal/runner"
)

func TestUnixSocketStatusRoundTrip(t *testing.T) {
	temp := t.TempDir()
	socket := filepath.Join(temp, "netrouterd.sock")
	server := New(network.NewManager(runner.OSExecutor{}))
	listener, err := server.ListenUnix(socket)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	info, err := os.Stat(socket)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o660 {
		t.Fatalf("socket mode = %o, want 660", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, listener) }()

	connection, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer connection.Close()
	request := protocol.Request{Version: protocol.Version, ID: "test-1", Method: protocol.GetSystemStatus}
	if err := json.NewEncoder(connection).Encode(request); err != nil {
		t.Fatalf("send request: %v", err)
	}
	var response protocol.Response
	if err := json.NewDecoder(connection).Decode(&response); err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("unexpected protocol error: %#v", response.Error)
	}
	var status protocol.SystemStatus
	if err := json.Unmarshal(response.Result, &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status.Identity == "" || status.Architecture == "" {
		t.Fatalf("unexpected status: %#v", status)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop when context was cancelled")
	}
}

func TestUnsupportedVersionReturnsProtocolError(t *testing.T) {
	server := New(network.NewManager(runner.OSExecutor{}))
	response := server.dispatch(context.Background(), protocol.Request{Version: 99, ID: "bad-version", Method: protocol.GetSystemStatus})
	if response.Error == nil || response.Error.Code != "unsupported_version" {
		t.Fatalf("unexpected response: %#v", response)
	}
}
