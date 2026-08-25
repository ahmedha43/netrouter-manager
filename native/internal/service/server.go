// Package service exposes the NetRouter protocol over a root-only Unix socket
// and an optional mutually-authenticated TLS listener for the Windows manager.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/ahmedha43/netrouter-manager/native/internal/network"
	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

type Server struct {
	network     *network.Manager
	connections sync.Map
}

func New(manager *network.Manager) *Server { return &Server{network: manager} }

func (s *Server) ListenUnix(path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create socket directory: %w", err)
	}
	if err := os.RemoveAll(path); err != nil {
		return nil, fmt.Errorf("remove stale socket: %w", err)
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen on Unix socket: %w", err)
	}
	if err := os.Chmod(path, 0o660); err != nil {
		listener.Close()
		return nil, fmt.Errorf("restrict socket permissions: %w", err)
	}
	return listener, nil
}

func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	var wait sync.WaitGroup
	defer wait.Wait()
	go func() {
		<-ctx.Done()
		_ = listener.Close()
		s.closeActiveConnections()
	}()
	for {
		connection, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("accept management connection: %w", err)
		}
		wait.Add(1)
		go func() { defer wait.Done(); s.handleConnection(ctx, connection) }()
	}
}

func (s *Server) handleConnection(ctx context.Context, connection net.Conn) {
	s.connections.Store(connection, struct{}{})
	defer func() {
		s.connections.Delete(connection)
		_ = connection.Close()
	}()
	decoder := json.NewDecoder(io.LimitReader(connection, 1<<20))
	encoder := json.NewEncoder(connection)
	for {
		var request protocol.Request
		if err := decoder.Decode(&request); err != nil {
			return
		}
		if err := encoder.Encode(s.dispatch(ctx, request)); err != nil {
			return
		}
	}
}

func (s *Server) closeActiveConnections() {
	s.connections.Range(func(key, _ any) bool {
		_ = key.(net.Conn).Close()
		return true
	})
}

func (s *Server) dispatch(ctx context.Context, request protocol.Request) protocol.Response {
	response := protocol.Response{Version: protocol.Version, ID: request.ID}
	if request.Version != protocol.Version {
		return errorResponse(response, "unsupported_version", "unsupported protocol version")
	}
	result, err := s.call(ctx, request)
	if err != nil {
		return errorResponse(response, "request_failed", err.Error())
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return errorResponse(response, "internal", "cannot encode response")
	}
	response.Result = encoded
	return response
}

func (s *Server) call(ctx context.Context, request protocol.Request) (any, error) {
	switch request.Method {
	case protocol.GetSystemStatus:
		return network.ReadSystemStatus()
	case protocol.SetIdentity:
		var params protocol.SetIdentityParams
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, fmt.Errorf("decode identity: %w", err)
		}
		return map[string]bool{"ok": true}, s.network.SetIdentity(ctx, params)
	case protocol.RebootSystem:
		var params protocol.RebootParams
		if len(request.Params) > 0 {
			_ = json.Unmarshal(request.Params, &params)
		}
		return map[string]bool{"ok": true}, s.network.RebootSystem(ctx, params)
	case protocol.GetSystemLogs:
		return network.ReadSystemLogs(), nil
	case protocol.ListInterfaces:
		return s.network.ListInterfaces()
	case protocol.GetTrafficStats:
		return network.ReadTrafficStats()
	case protocol.SetLinkState:
		var params protocol.SetLinkStateParams
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, fmt.Errorf("decode link state: %w", err)
		}
		return map[string]bool{"ok": true}, s.network.SetLinkState(ctx, params)
	case protocol.AssignAddress:
		var params protocol.AssignAddressParams
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, fmt.Errorf("decode address: %w", err)
		}
		return map[string]bool{"ok": true}, s.network.AssignAddress(ctx, params)
	case protocol.ReplaceRoute:
		var params protocol.ReplaceDefaultRouteParams
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, fmt.Errorf("decode route: %w", err)
		}
		return map[string]bool{"ok": true}, s.network.ReplaceDefaultRoute(ctx, params)
	case protocol.ApplyDHCPDNS:
		var params protocol.DHCPDNSParams
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, fmt.Errorf("decode DHCP/DNS configuration: %w", err)
		}
		return map[string]bool{"ok": true}, s.network.ApplyDHCPDNS(ctx, params)
	case protocol.ListDHCPLeases:
		return s.network.ListDHCPLeases()
	case protocol.ApplyFirewall:
		var params protocol.FirewallParams
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, fmt.Errorf("decode firewall configuration: %w", err)
		}
		return map[string]bool{"ok": true}, s.network.ApplyFirewall(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported method %q", request.Method)
	}
}

func errorResponse(response protocol.Response, code, message string) protocol.Response {
	response.Error = &protocol.APIError{Code: code, Message: message}
	return response
}
