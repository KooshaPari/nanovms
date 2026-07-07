package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kooshapari/nanovms/internal/domain"
	"github.com/kooshapari/nanovms/internal/token"
)

// fakePort is a minimal in-memory SandboxPort stub for API routing tests.
type fakePort struct {
	s []domain.Sandbox
}

func (f *fakePort) Create(ctx context.Context, cfg domain.SandboxConfig) (*domain.Sandbox, error) {
	if cfg.Name == "" {
		cfg.Name = "svc"
	}
	sb := &domain.Sandbox{ID: cfg.Name, Name: cfg.Name, Status: domain.SandboxStatusRunning}
	f.s = append(f.s, *sb)
	return sb, nil
}
func (f *fakePort) Start(ctx context.Context, id string) error           { return nil }
func (f *fakePort) Stop(ctx context.Context, id string, force bool) error { return nil }
func (f *fakePort) Delete(ctx context.Context, id string) error {
	f.s = nil
	return nil
}
func (f *fakePort) List(ctx context.Context) ([]*domain.Sandbox, error) {
	out := make([]*domain.Sandbox, 0, len(f.s))
	for i := range f.s {
		s := f.s[i]
		out = append(out, &s)
	}
	return out, nil
}
func (f *fakePort) Get(ctx context.Context, id string) (*domain.Sandbox, error) {
	for _, sb := range f.s {
		if sb.ID == id {
			s := sb
			return &s, nil
		}
	}
	return nil, errors.New("not found")
}
func (f *fakePort) Logs(ctx context.Context, id string, follow bool) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (f *fakePort) Exec(ctx context.Context, id string, cmd []string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (f *fakePort) Metrics(ctx context.Context, id string) (*domain.SandboxMetrics, error) {
	return &domain.SandboxMetrics{}, nil
}

func withAuth(t *testing.T, tok *token.Manager, req *http.Request) *http.Request {
	t.Helper()
	if tok.Len() == 0 {
		t.Fatal("token manager empty")
	}
	req.Header.Set("Authorization", "Bearer deadbeef")
	return req
}

func TestHealthz_Readyz_NoAuth(t *testing.T) {
	r := NewRouter(Handlers{Token: nil, Port: nil})
	for _, path := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s => %d", path, rr.Code)
		}
	}
}

func TestAuthEnforced(t *testing.T) {
	m, _ := token.NewFromHex([]string{"deadbeef"})
	r := NewRouter(Handlers{Token: m, Port: (&fakePort{})})
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("GET /v1/models without auth = %d, want 401", rr.Code)
	}

	req = withAuth(t, m, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /v1/models with auth = %d, want 200", rr.Code)
	}
}

func TestDeployAndList(t *testing.T) {
	m, _ := token.NewFromHex([]string{"deadbeef"})
	p := &fakePort{}
	r := NewRouter(Handlers{Token: m, Port: p})

	deploy := `{"name":"test", "type":"container", "engine":"dummy"}`
	req := withAuth(t, m, httptest.NewRequest(http.MethodPost, "/v1/deploy", strings.NewReader(deploy)))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("deploy = %d", rr.Code)
	}

	req = withAuth(t, m, httptest.NewRequest(http.MethodGet, "/v1/sandboxes", nil))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list = %d", rr.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("list missing data")
	}
}
