package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// NewClient validation tests
// ---------------------------------------------------------------------------

func TestNewClientRejectsHTTP(t *testing.T) {
	t.Parallel()
	_, err := NewClient("http://example.com", "token", "user")
	if err == nil {
		t.Fatal("expected error for http:// baseURL, got nil")
	}
}

func TestNewClientRejectsEmpty(t *testing.T) {
	t.Parallel()
	_, err := NewClient("", "token", "user")
	if err == nil {
		t.Fatal("expected error for empty baseURL, got nil")
	}
}

func TestNewClientAcceptsHTTPS(t *testing.T) {
	t.Parallel()
	c, err := NewClient("https://example.com", "token", "user")
	if err != nil {
		t.Fatalf("unexpected error for https:// baseURL: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil *Client")
	}
}

func TestNewClientStripsTrailingSlash(t *testing.T) {
	t.Parallel()
	c, err := NewClient("https://example.com/", "token", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.HasSuffix(c.baseURL, "/") {
		t.Errorf("baseURL should not have a trailing slash, got %q", c.baseURL)
	}
}

// ---------------------------------------------------------------------------
// Helpers shared by the do/Get/Post tests
// ---------------------------------------------------------------------------

// newTestClientPointing creates a Client whose baseURL points at the given
// httptest.Server. The server's scheme is "http", so we bypass NewClient's
// HTTPS enforcement by constructing the struct directly.
func newTestClientPointing(srv *httptest.Server, token string) *Client {
	return &Client{
		baseURL: srv.URL,
		token:   token,
		userID:  "uid-test",
		http:    srv.Client(),
	}
}

// ---------------------------------------------------------------------------
// do() header tests
// ---------------------------------------------------------------------------

func TestDoSetsAuthHeader(t *testing.T) {
	t.Parallel()

	const token = "my-secret-token"
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, token)
	_, _ = c.do(http.MethodGet, "/path", nil)

	want := "Bearer " + token
	if capturedAuth != want {
		t.Errorf("Authorization header: got %q, want %q", capturedAuth, want)
	}
}

func TestDoSetsXRequestedWith(t *testing.T) {
	t.Parallel()

	var capturedHeader string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Get("X-Requested-With")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	_, _ = c.do(http.MethodGet, "/path", nil)

	const want = "XMLHttpRequest"
	if capturedHeader != want {
		t.Errorf("X-Requested-With: got %q, want %q", capturedHeader, want)
	}
}

// ---------------------------------------------------------------------------
// do() error-handling test
// ---------------------------------------------------------------------------

func TestDoReturnsErrorOnNon2xx(t *testing.T) {
	t.Parallel()

	const token = "super-secret-token"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized) // 401
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, token)
	_, err := c.do(http.MethodGet, "/path", nil)

	if err == nil {
		t.Fatal("expected an error for 401 response, got nil")
	}

	// Security requirement: token must not appear in the error message.
	if strings.Contains(err.Error(), token) {
		t.Errorf("error message must not contain the token; got: %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Get / Post helper tests
// ---------------------------------------------------------------------------

func TestGetHelper(t *testing.T) {
	t.Parallel()

	var capturedMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClientPointing(srv, "tok")
	_, err := c.Get("/resource")
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}

	if capturedMethod != http.MethodGet {
		t.Errorf("expected method GET, got %q", capturedMethod)
	}
}

func TestPostHelper(t *testing.T) {
	t.Parallel()

	var capturedMethod string
	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	const payload = `{"key":"value"}`
	c := newTestClientPointing(srv, "tok")
	_, err := c.Post("/resource", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("Post returned unexpected error: %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Errorf("expected method POST, got %q", capturedMethod)
	}
	if string(capturedBody) != payload {
		t.Errorf("expected body %q, got %q", payload, string(capturedBody))
	}
}
