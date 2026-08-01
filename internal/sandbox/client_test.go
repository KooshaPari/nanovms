package sandbox

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func listRequest(t *testing.T, client *Client) *http.Request {
	t.Helper()
	var got *http.Request
	client.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		got = req
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"object":"list","data":[]}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})

	if _, err := client.ListSandboxes(t.Context()); err != nil {
		t.Fatalf("ListSandboxes() error = %v", err)
	}
	if got == nil {
		t.Fatal("transport did not receive a request")
	}
	return got
}

func TestClientWithTokenAuthenticatesListRequests(t *testing.T) {
	got := listRequest(t, NewClientWithToken("/tmp/nvms.sock", "daemon-secret"))
	if want := "Bearer daemon-secret"; got.Header.Get("Authorization") != want {
		t.Fatalf("Authorization = %q, want %q", got.Header.Get("Authorization"), want)
	}
}

func TestClientWithoutTokenLeavesAuthorizationUnset(t *testing.T) {
	got := listRequest(t, NewClient("/tmp/nvms.sock"))
	if got.Header.Get("Authorization") != "" {
		t.Fatalf("Authorization = %q, want unset", got.Header.Get("Authorization"))
	}
}
