// Package manager implements the small NetRouter protocol client used by the
// Windows EXE. It never shells out and it always uses a configured TLS socket.
package manager

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

type Client struct {
	connection net.Conn
	sequence   uint64
}

func DialTLS(address string, config *tls.Config) (*Client, error) {
	connection, err := tls.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("connect securely to router: %w", err)
	}
	return &Client{connection: connection}, nil
}
func (c *Client) Close() error { return c.connection.Close() }
func (c *Client) Call(ctx context.Context, method protocol.Method, params any, result any) error {
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.connection.SetDeadline(deadline)
	} else {
		_ = c.connection.SetDeadline(time.Now().Add(8 * time.Second))
	}
	defer c.connection.SetDeadline(time.Time{})
	encoded, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("encode request parameters: %w", err)
	}
	request := protocol.Request{Version: protocol.Version, ID: fmt.Sprintf("%d", atomic.AddUint64(&c.sequence, 1)), Method: method, Params: encoded}
	if err := json.NewEncoder(c.connection).Encode(request); err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	var response protocol.Response
	if err := json.NewDecoder(c.connection).Decode(&response); err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if response.ID != request.ID {
		return fmt.Errorf("response correlation mismatch")
	}
	if response.Error != nil {
		return fmt.Errorf("router error %s: %s", response.Error.Code, response.Error.Message)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(response.Result, result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
