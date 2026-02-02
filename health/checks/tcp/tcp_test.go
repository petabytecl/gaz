package tcp_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/petabytecl/gaz/health/checks/tcp"
)

func TestNew_EmptyAddress(t *testing.T) {
	check := tcp.New(tcp.Config{})
	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error for empty address")
	}
	if got := err.Error(); got != "tcp: address is empty" {
		t.Errorf("got %q, want %q", got, "tcp: address is empty")
	}
}

func TestNew_SuccessfulConnection(t *testing.T) {
	// Create a test TCP server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create test listener: %v", err)
	}
	defer ln.Close()

	// Accept connections in the background
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	check := tcp.New(tcp.Config{
		Addr: ln.Addr().String(),
	})

	err = check(context.Background())
	if err != nil {
		t.Errorf("expected successful connection, got error: %v", err)
	}
}

func TestNew_ConnectionFailure(t *testing.T) {
	// Use a port that's unlikely to be listening
	check := tcp.New(tcp.Config{
		Addr:    "127.0.0.1:1", // Port 1 is typically not available
		Timeout: 100 * time.Millisecond,
	})

	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error for connection to invalid port")
	}
}

func TestNew_ContextCancellation(t *testing.T) {
	// Create a test TCP server that accepts slowly
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create test listener: %v", err)
	}
	defer ln.Close()

	// Don't accept any connections, just let them time out

	check := tcp.New(tcp.Config{
		Addr:    ln.Addr().String(),
		Timeout: 5 * time.Second, // Long timeout
	})

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = check(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestNew_DefaultTimeout(t *testing.T) {
	// Just verify it doesn't panic with default timeout
	check := tcp.New(tcp.Config{
		Addr: "127.0.0.1:1",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Should fail (either connection refused or timeout), but not panic
	_ = check(ctx)
}
