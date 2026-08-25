// Package runner isolates controlled operating-system command execution.
// Network mutations are only reachable through validated manager methods.
package runner

import "context"

type Executor interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}
