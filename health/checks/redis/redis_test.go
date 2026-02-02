package redis

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPinger is a mock implementation of Pinger for testing.
type mockPinger struct {
	result string
	err    error
}

func (m *mockPinger) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	if m.err != nil {
		cmd.SetErr(m.err)
	} else {
		cmd.SetVal(m.result)
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
	client := &mockPinger{result: "PONG"}
	ctx := context.Background()

	check := New(Config{Client: client})
	err := check(ctx)

	assert.NoError(t, err)
}

func TestNew_PingFailure(t *testing.T) {
	client := &mockPinger{err: errors.New("connection refused")}
	ctx := context.Background()

	check := New(Config{Client: client})
	err := check(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestNew_UnexpectedResponse(t *testing.T) {
	client := &mockPinger{result: "UNEXPECTED"}
	ctx := context.Background()

	check := New(Config{Client: client})
	err := check(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected ping response")
	assert.Contains(t, err.Error(), "UNEXPECTED")
}

func TestNew_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	client := &mockPinger{err: ctx.Err()}

	check := New(Config{Client: client})
	err := check(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")
}

// Verify return type matches health.CheckFunc signature.
var _ func(context.Context) error = New(Config{})
