package linux

import (
	"context"
	"io"
	"os/exec"
)

type sandboxSyscalls interface {
	LookPath(file string) (string, error)
	Start(ctx context.Context, name string, args ...string) (int, error)
	Run(ctx context.Context, name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error
}

type execSandboxSyscalls struct{}

func (execSandboxSyscalls) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (execSandboxSyscalls) Start(ctx context.Context, name string, args ...string) (int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	return cmd.Process.Pid, nil
}

func (execSandboxSyscalls) Run(
	ctx context.Context,
	name string,
	args []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
