package dns_test

import (
	"context"
	"testing"
	"time"

	"github.com/petabytecl/gaz/health/checks/dns"
)

func TestNew_EmptyHostname(t *testing.T) {
	check := dns.New(dns.Config{})
	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error for empty hostname")
	}
	if got := err.Error(); got != "dns: hostname is empty" {
		t.Errorf("got %q, want %q", got, "dns: hostname is empty")
	}
}

func TestNew_SuccessfulResolution(t *testing.T) {
	// localhost should always resolve
	check := dns.New(dns.Config{
		Host: "localhost",
	})

	err := check(context.Background())
	if err != nil {
		t.Errorf("expected successful resolution for localhost, got error: %v", err)
	}
}

func TestNew_ResolutionFailure(t *testing.T) {
	// Use a domain that should not exist
	check := dns.New(dns.Config{
		Host:    "this-domain-should-not-exist-xyz123456.invalid",
		Timeout: 1 * time.Second,
	})

	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error for non-existent domain")
	}
}

func TestNew_ContextTimeout(t *testing.T) {
	check := dns.New(dns.Config{
		Host:    "localhost",
		Timeout: 5 * time.Second,
	})

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := check(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestNew_DefaultTimeout(t *testing.T) {
	// Just verify it doesn't panic with default timeout
	check := dns.New(dns.Config{
		Host: "localhost",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should work (localhost resolves quickly)
	_ = check(ctx)
}
