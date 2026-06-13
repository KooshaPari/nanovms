package linux

import (
	"context"
	"errors"
	"io"
	"reflect"
	"syscall"
	"testing"

	"go.uber.org/mock/gomock"
)

type mockSandboxSyscalls struct {
	ctrl     *gomock.Controller
	recorder *mockSandboxSyscallsRecorder
}

type mockSandboxSyscallsRecorder struct {
	mock *mockSandboxSyscalls
}

func newMockSandboxSyscalls(ctrl *gomock.Controller) *mockSandboxSyscalls {
	mock := &mockSandboxSyscalls{ctrl: ctrl}
	mock.recorder = &mockSandboxSyscallsRecorder{mock: mock}
	return mock
}

func (m *mockSandboxSyscalls) EXPECT() *mockSandboxSyscallsRecorder {
	return m.recorder
}

func (m *mockSandboxSyscalls) LookPath(file string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LookPath", file)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (r *mockSandboxSyscallsRecorder) LookPath(file any) *gomock.Call {
	r.mock.ctrl.T.Helper()
	return r.mock.ctrl.RecordCallWithMethodType(
		r.mock,
		"LookPath",
		reflect.TypeOf((*mockSandboxSyscalls)(nil).LookPath),
		file,
	)
}

func (m *mockSandboxSyscalls) Start(ctx context.Context, name string, args ...string) (int, error) {
	m.ctrl.T.Helper()
	callArgs := []any{ctx, name}
	for _, arg := range args {
		callArgs = append(callArgs, arg)
	}
	ret := m.ctrl.Call(m, "Start", callArgs...)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (r *mockSandboxSyscallsRecorder) Start(ctx, name any, args ...any) *gomock.Call {
	r.mock.ctrl.T.Helper()
	callArgs := []any{ctx, name}
	callArgs = append(callArgs, args...)
	return r.mock.ctrl.RecordCallWithMethodType(
		r.mock,
		"Start",
		reflect.TypeOf((*mockSandboxSyscalls)(nil).Start),
		callArgs...,
	)
}

func (m *mockSandboxSyscalls) Run(
	ctx context.Context,
	name string,
	args []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", ctx, name, args, stdin, stdout, stderr)
	ret0, _ := ret[0].(error)
	return ret0
}

func (r *mockSandboxSyscallsRecorder) Run(ctx, name, args, stdin, stdout, stderr any) *gomock.Call {
	r.mock.ctrl.T.Helper()
	return r.mock.ctrl.RecordCallWithMethodType(
		r.mock,
		"Run",
		reflect.TypeOf((*mockSandboxSyscalls)(nil).Run),
		ctx, name, args, stdin, stdout, stderr,
	)
}

func TestSyscallSmokeNamespaceRunnerOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSyscalls := newMockSandboxSyscalls(ctrl)
	adapter := newWithSyscalls(Native, true, mockSyscalls)
	ctx := context.Background()
	var stdoutBuffer noopWriter
	var stderrBuffer noopWriter

	gomock.InOrder(
		mockSyscalls.EXPECT().LookPath(gomock.Eq("ip")).Times(1).Return("/usr/bin/ip", nil),
		mockSyscalls.EXPECT().LookPath(gomock.Eq("unshare")).Times(1).Return("/usr/bin/unshare", nil),
		mockSyscalls.EXPECT().LookPath(gomock.Eq("mount")).Times(1).Return("/usr/bin/mount", nil),
		mockSyscalls.EXPECT().
			Start(
				gomock.Any(),
				gomock.Eq("unshare"),
				gomock.Eq("--user"),
				gomock.Eq("--map-root-user"),
				gomock.Eq("sleep"),
				gomock.Eq("infinity"),
			).
			Times(1).
			Return(4242, nil),
		mockSyscalls.EXPECT().
			Run(
				gomock.Any(),
				gomock.Eq("unshare"),
				gomock.Eq([]string{"--user", "--map-root-user", "--mount", "--ipc", "--pid", "--fork", "bash", "-c", "echo smoke"}),
				gomock.Nil(),
				gomock.AssignableToTypeOf(io.Writer(&stdoutBuffer)),
				gomock.AssignableToTypeOf(io.Writer(&stderrBuffer)),
			).
			Times(1).
			Return(nil),
	)

	if err := adapter.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	id, err := adapter.Create(ctx, "smoke")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if id != "devenv-smoke" {
		t.Fatalf("unexpected sandbox id: %s", id)
	}

	if err := adapter.Exec(ctx, id, []string{"echo", "smoke"}, nil, &stdoutBuffer, &stderrBuffer); err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
}

func TestSyscallSmokeCreateNativePropagatesENOSYS(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSyscalls := newMockSandboxSyscalls(ctrl)
	adapter := newWithSyscalls(Native, true, mockSyscalls)
	ctx := context.Background()

	mockSyscalls.EXPECT().
		Start(
			gomock.Any(),
			gomock.Eq("unshare"),
			gomock.Eq("--user"),
			gomock.Eq("--map-root-user"),
			gomock.Eq("sleep"),
			gomock.Eq("infinity"),
		).
		Times(1).
		Return(0, syscall.ENOSYS)

	_, err := adapter.createNative(ctx, "smoke")
	if !errors.Is(err, syscall.ENOSYS) {
		t.Fatalf("expected ENOSYS, got %v", err)
	}
}

func TestSyscallSmokeExecNativePropagatesEPERM(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSyscalls := newMockSandboxSyscalls(ctrl)
	adapter := newWithSyscalls(Native, true, mockSyscalls)
	ctx := context.Background()
	var stdoutBuffer noopWriter
	var stderrBuffer noopWriter

	mockSyscalls.EXPECT().
		Run(
			gomock.Any(),
			gomock.Eq("unshare"),
			gomock.Eq([]string{"--user", "--map-root-user", "--mount", "--ipc", "--pid", "--fork", "bash", "-c", "echo fail"}),
			gomock.Nil(),
			gomock.AssignableToTypeOf(io.Writer(&stdoutBuffer)),
			gomock.AssignableToTypeOf(io.Writer(&stderrBuffer)),
		).
		Times(1).
		Return(syscall.EPERM)

	err := adapter.execNative(ctx, "devenv-fail", []string{"echo", "fail"}, nil, &stdoutBuffer, &stderrBuffer)
	if !errors.Is(err, syscall.EPERM) {
		t.Fatalf("expected EPERM, got %v", err)
	}
}

type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
