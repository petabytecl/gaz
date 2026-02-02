package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient is a minimal mock for redis.UniversalClient.
// We embed UniversalClient to satisfy the interface, then override Ping.
type mockClient struct {
	redis.UniversalClient
	pingErr  error
	pingResp string
}

func (m *mockClient) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	if m.pingErr != nil {
		cmd.SetErr(m.pingErr)
	} else {
		cmd.SetVal(m.pingResp)
	}
	return cmd
}

func TestNew_NilClient(t *testing.T) {
	check := New(Config{Client: nil})
	err := check(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "client is nil")
}

func TestNew_SuccessfulPing(t *testing.T) {
	client := &mockClient{pingResp: "PONG"}

	check := New(Config{Client: client})
	err := check(context.Background())

	assert.NoError(t, err)
}

func TestNew_PingFailure(t *testing.T) {
	client := &mockClient{pingErr: errors.New("connection refused")}

	check := New(Config{Client: client})
	err := check(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestNew_UnexpectedResponse(t *testing.T) {
	client := &mockClient{pingResp: "UNEXPECTED"}

	check := New(Config{Client: client})
	err := check(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected ping response")
	assert.Contains(t, err.Error(), "UNEXPECTED")
}

func TestNew_ContextCancellation(t *testing.T) {
	// Create a mock that respects context cancellation
	client := &mockClientWithDelay{delay: 5 * time.Second, pingResp: "PONG"}

	check := New(Config{Client: client})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := check(ctx)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")
	// Ensure it returned quickly due to context cancellation, not the full 5s delay
	assert.Less(t, elapsed, time.Second, "should cancel quickly")
}

// mockClientWithDelay adds delay support for context cancellation testing.
type mockClientWithDelay struct {
	redis.UniversalClient
	delay    time.Duration
	pingErr  error
	pingResp string
}

func (m *mockClientWithDelay) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)

	if m.delay > 0 {
		select {
		case <-ctx.Done():
			cmd.SetErr(ctx.Err())
			return cmd
		case <-time.After(m.delay):
		}
	}

	if m.pingErr != nil {
		cmd.SetErr(m.pingErr)
	} else {
		cmd.SetVal(m.pingResp)
	}
	return cmd
}

// Verify return type matches health.CheckFunc signature.
var _ func(context.Context) error = New(Config{})
