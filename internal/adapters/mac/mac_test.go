package mac

import (
	"strings"
	"testing"

	"github.com/kooshapari/nanovms/internal/ports"
)

func newWith(lima, hyperkit, firecracker string) *Adapter {
	return &Adapter{
		limaPath:        lima,
		hyperkitPath:    hyperkit,
		firecrackerPath: firecracker,
	}
}

func TestName_FirecrackerPreferred(t *testing.T) {
	a := newWith("/usr/local/bin/limactl", "/usr/local/bin/hyperkit", "/usr/local/bin/firecracker")
	if got := a.Name(); got != "firecracker" {
		t.Fatalf("want firecracker (preferred), got %q", got)
	}
}

func TestName_HyperKit(t *testing.T) {
	a := newWith("/usr/local/bin/limactl", "/usr/local/bin/hyperkit", "")
	if got := a.Name(); got != "hyperkit" {
		t.Fatalf("want hyperkit, got %q", got)
	}
}

func TestName_Lima(t *testing.T) {
	a := newWith("/usr/local/bin/limactl", "", "")
	if got := a.Name(); got != "lima" {
		t.Fatalf("want lima, got %q", got)
	}
}

func TestName_Colima(t *testing.T) {
	a := newWith("/usr/local/bin/colima", "", "")
	if got := a.Name(); got != "colima" {
		t.Fatalf("want colima (path substring), got %q", got)
	}
}

func TestName_Unknown(t *testing.T) {
	a := &Adapter{}
	if got := a.Name(); got != "unknown" {
		t.Fatalf("want unknown, got %q", got)
	}
}

func TestSupportedTiers_AllPresent(t *testing.T) {
	a := newWith("/usr/local/bin/limactl", "/usr/local/bin/hyperkit", "/usr/local/bin/firecracker")
	got := a.SupportedTiers()
	want := []ports.VMTier{ports.VMTierNative, ports.VMTierLimaVZ, ports.VMTierMicroVM}
	if len(got) != len(want) {
		t.Fatalf("len mismatch: want %d tiers, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tier %d: want %v, got %v", i, want[i], got[i])
		}
	}
}

func TestSupportedTiers_FirecrackerOnly(t *testing.T) {
	a := newWith("", "", "/usr/local/bin/firecracker")
	got := a.SupportedTiers()
	if len(got) != 1 || got[0] != ports.VMTierMicroVM {
		t.Fatalf("want [MicroVM], got %v", got)
	}
}

func TestSupportedTiers_Empty(t *testing.T) {
	a := &Adapter{}
	if got := a.SupportedTiers(); len(got) != 0 {
		t.Fatalf("want empty tiers, got %v", got)
	}
}

func TestAdapterStructFields(t *testing.T) {
	a := newWith("/p/lima", "/p/hk", "/p/fc")
	if a.limaPath != "/p/lima" || a.hyperkitPath != "/p/hk" || a.firecrackerPath != "/p/fc" {
		t.Fatalf("struct fields not preserved: %+v", a)
	}
}

func TestName_ColimaPathDetection(t *testing.T) {
	// Path contains 'colima' but not 'limactl'; should still resolve to colima.
	a := newWith("/opt/homebrew/bin/colima", "", "")
	if got := a.Name(); !strings.Contains(got, "colima") {
		t.Fatalf("colima path should yield colima name, got %q", got)
	}
}
