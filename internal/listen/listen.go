package listen

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// Listener wraps a net.Listener with lifecycle helpers.
type Listener struct {
	ln        net.Listener
	cleanup   func() error
	errCh     chan error
	closedCh  chan struct{}
	closeOnce sync.Once
}

// NewUDS creates/returns a Unix domain listener at socketPath.
//
// If socketPath is absolute, it is used as-is.
// If socketPath is relative, it is joined with runBase.
func NewUDS(ctx context.Context, socketPath, runBase string) (*Listener, error) {
	if socketPath == "" {
		socketPath = "routed.sock"
	}
	if runBase == "" {
		runBase = "/tmp"
	}
	if !filepath.IsAbs(socketPath) {
		socketPath = filepath.Join(runBase, socketPath)
	}

	if err := os.MkdirAll(filepath.Dir(socketPath), 0o750); err != nil {
		return nil, fmt.Errorf("listen: mkdir failed: %w", err)
	}
	_ = os.Remove(socketPath)

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen: bind failed: %w", err)
	}
	if err := os.Chmod(socketPath, 0o660); err != nil {
		_ = ln.Close()
		_ = os.Remove(socketPath)
		return nil, fmt.Errorf("listen: chmod failed: %w", err)
	}

	listener := &Listener{
		ln:       ln,
		errCh:    make(chan error, 1),
		closedCh: make(chan struct{}),
	}
	listener.cleanup = func() error { return os.Remove(socketPath) }
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	return listener, nil
}

// NewTCP creates a TCP listener on addr.
//
// If tlsCfg is nil, the listener is plain TCP.
// addr defaults to ":8443" if empty.
func NewTCP(ctx context.Context, addr string, tlsCfg *tls.Config) (*Listener, error) {
	if addr == "" {
		addr = ":8443"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen: tcp bind %q failed: %w", addr, err)
	}

	var tlsLn = ln
	if tlsCfg != nil {
		tlsLn = tls.NewListener(ln, tlsCfg)
	}

	listener := &Listener{
		ln:       tlsLn,
		errCh:    make(chan error, 1),
		closedCh: make(chan struct{}),
	}
	listener.cleanup = func() error { return nil } // no file to remove
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	return listener, nil
}

// Serve runs HTTP on the listener.
func (l *Listener) Serve(s *http.Server) error {
	err := s.Serve(l.ln)
	select {
	case l.errCh <- err:
	default:
	}
	close(l.closedCh)
	return err
}

// Close closes socket and removes file.
func (l *Listener) Close() error {
	l.closeOnce.Do(func() {
		_ = l.ln.Close()
		_ = l.cleanup()
		close(l.errCh)
		close(l.closedCh)
	})
	return nil
}

// ErrCh returns serve errors asynchronously.
func (l *Listener) ErrCh() <-chan error { return l.errCh }

// ClosedCh returns a channel that closes when listener is torn down.
func (l *Listener) ClosedCh() <-chan struct{} { return l.closedCh }
