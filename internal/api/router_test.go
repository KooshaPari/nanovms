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
	s          []domain.Sandbox
	startErr   error
	startCalls []string
}

func (f *fakePort) Probe(context.Context) error { return nil }

func (f *fakePort) Create(ctx context.Context, cfg domain.SandboxConfig) (*domain.Sandbox, error) {
	if cfg.Name == "" {
		cfg.Name = "svc"
	}
	sb := &domain.Sandbox{ID: cfg.Name, Name: cfg.Name, Status: domain.SandboxStatusPending}
	f.s = append(f.s, *sb)
	return sb, nil
}
func (f *fakePort) Start(ctx context.Context, id string) error {
	f.startCalls = append(f.startCalls, id)
	if f.startErr != nil {
		return f.startErr
	}
	for i := range f.s {
		if f.s[i].ID == id {
			f.s[i].Status = domain.SandboxStatusRunning
		}
	}
	return nil
}
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
	t.Run("liveness does not require provider", func(t *testing.T) {
		r := NewRouter(Handlers{Token: nil, Port: nil})
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("/healthz => %d", rr.Code)
		}
	})

	t.Run("readiness fails closed without provider", func(t *testing.T) {
		r := NewRouter(Handlers{Token: nil, Port: nil})
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("/readyz => %d, want %d", rr.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("readiness probes provider", func(t *testing.T) {
		r := NewRouter(Handlers{Token: nil, Port: &fakePort{}})
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("/readyz => %d, want 200", rr.Code)
		}
	})
}

func TestAuditMiddlewareRecordsStatusAndRequestID(t *testing.T) {
	audit := NewAuditLogger("")
	r := NewRouter(Handlers{Port: nil, AuditLog: audit})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Request-ID", "pilot-123")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("healthz = %d, want 200", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz = %d, want 503", rr.Code)
	}

	entries := audit.Query("", "", "", "", 10, 0)
	if len(entries) != 2 {
		t.Fatalf("audit entries = %d, want 2", len(entries))
	}
	if entries[0].StatusCode != http.StatusServiceUnavailable || entries[1].StatusCode != http.StatusOK {
		t.Fatalf("audit statuses = %d, %d, want 503, 200", entries[0].StatusCode, entries[1].StatusCode)
	}
	if entries[1].RequestID != "pilot-123" || entries[0].RequestID == "" {
		t.Fatalf("audit request IDs = %q, %q", entries[0].RequestID, entries[1].RequestID)
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
	if len(p.startCalls) != 1 || p.startCalls[0] != "test" {
		t.Fatalf("start calls = %#v, want [test]", p.startCalls)
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

func TestDeployStartFailureReturnsFailedSandbox(t *testing.T) {
	m, _ := token.NewFromHex([]string{"deadbeef"})
	p := &fakePort{startErr: errors.New("adapter capability unavailable: runsc not found")}
	r := NewRouter(Handlers{Token: m, Port: p})

	req := withAuth(t, m, httptest.NewRequest(http.MethodPost, "/v1/deploy", strings.NewReader(`{"name":"needs-runsc"}`)))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("deploy start failure = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	var body struct {
		Error   string               `json:"error"`
		Status  domain.SandboxStatus `json:"status"`
		Sandbox domain.Sandbox       `json:"sandbox"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode failure response: %v", err)
	}
	if body.Error == "" || !strings.Contains(body.Error, "runsc not found") {
		t.Fatalf("failure response error = %q", body.Error)
	}
	if body.Status != domain.SandboxStatusFailed || body.Sandbox.Status != domain.SandboxStatusFailed {
		t.Fatalf("failure statuses = %q/%q, want failed", body.Status, body.Sandbox.Status)
	}
}
