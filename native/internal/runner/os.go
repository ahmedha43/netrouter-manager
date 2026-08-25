package runner

import (
	"context"
	"os/exec"
)

type OSExecutor struct{}

func (OSExecutor) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}
