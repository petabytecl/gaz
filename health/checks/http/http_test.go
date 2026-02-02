package httpcheck_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpcheck "github.com/petabytecl/gaz/health/checks/http"
)

func TestNew_EmptyURL(t *testing.T) {
	check := httpcheck.New(httpcheck.Config{})
	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
	if err.Error() != "http: URL is empty" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestNew_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	check := httpcheck.New(httpcheck.Config{URL: server.URL})
	err := check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNew_CustomExpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	check := httpcheck.New(httpcheck.Config{
		URL:                server.URL,
		ExpectedStatusCode: http.StatusNoContent,
	})
	err := check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNew_UnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	check := httpcheck.New(httpcheck.Config{URL: server.URL})
	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error for unexpected status code")
	}
	expected := "http: unexpected status 503 (expected 200)"
	if err.Error() != expected {
		t.Errorf("got %q, want %q", err.Error(), expected)
	}
}

func TestNew_ConnectionFailure(t *testing.T) {
	// Use a port that should not have anything listening
	check := httpcheck.New(httpcheck.Config{
		URL:     "http://127.0.0.1:1", // Port 1 is privileged and unlikely to be used
		Timeout: 100 * time.Millisecond,
	})
	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestNew_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second) // Slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	check := httpcheck.New(httpcheck.Config{URL: server.URL})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := check(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestNew_CustomClient(t *testing.T) {
	requestReceived := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	customClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	check := httpcheck.New(httpcheck.Config{
		URL:    server.URL,
		Client: customClient,
	})
	err := check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !requestReceived {
		t.Error("request was not made with custom client")
	}
}

func TestNew_DoesNotFollowRedirects(t *testing.T) {
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer redirectTarget.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget.URL, http.StatusMovedPermanently)
	}))
	defer server.Close()

	// With default behavior (don't follow redirects), we expect 301
	check := httpcheck.New(httpcheck.Config{
		URL:                server.URL,
		ExpectedStatusCode: http.StatusMovedPermanently,
	})
	err := check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNew_ConnectionCloseHeader(t *testing.T) {
	var receivedConnection string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedConnection = r.Header.Get("Connection")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	check := httpcheck.New(httpcheck.Config{URL: server.URL})
	err := check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedConnection != "close" {
		t.Errorf("expected Connection: close header, got %q", receivedConnection)
	}
}
