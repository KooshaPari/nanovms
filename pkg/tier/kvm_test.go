// SPDX-License-Identifier: MIT OR Apache-2.0
package tier

import (
	"context"
	"errors"
	"testing"

	"github.com/kooshapari/nanovms/internal/domain"
)

func TestKVMStartRejectsUnimplementedGuestLifecycle(t *testing.T) {
	adapter := NewKVMAdapter()
	if err := adapter.Start(context.Background(), "unstarted-guest"); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("Start must reject an unimplemented VM launch, got %v", err)
	}
}

func TestKVMDeployDoesNotReportAnUnstartedGuest(t *testing.T) {
	sandbox, err := NewKVMAdapter().Deploy(context.Background(), domain.SandboxConfig{Name: "unstarted"})
	if sandbox != nil || !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("Deploy must reject its missing VM implementation, got sandbox=%v error=%v", sandbox, err)
	}
}
