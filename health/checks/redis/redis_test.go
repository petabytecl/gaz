package redis

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

func TestNew_NilClient(t *testing.T) {
	check := New(Config{Client: nil})
	err := check(context.Background())

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilClient)
}

func TestNew_SuccessfulPing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock.NewClient(ctrl)
	client.EXPECT().
		Do(gomock.Any(), mock.Match("PING")).
		Return(mock.Result(mock.ValkeyString("PONG")))

	ctx := context.Background()
	check := New(Config{Client: client})
	err := check(ctx)

	assert.NoError(t, err)
}

func TestNew_PingFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock.NewClient(ctrl)
	client.EXPECT().
		Do(gomock.Any(), mock.Match("PING")).
		Return(mock.ErrorResult(errors.New("connection refused")))

	ctx := context.Background()
	check := New(Config{Client: client})
	err := check(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestNew_UnexpectedResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock.NewClient(ctrl)
	client.EXPECT().
		Do(gomock.Any(), mock.Match("PING")).
		Return(mock.Result(mock.ValkeyString("UNEXPECTED")))

	ctx := context.Background()
	check := New(Config{Client: client})
	err := check(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected ping response")
	assert.Contains(t, err.Error(), "UNEXPECTED")
}

func TestNew_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	client := mock.NewClient(ctrl)
	client.EXPECT().
		Do(gomock.Any(), mock.Match("PING")).
		Return(mock.ErrorResult(ctx.Err()))

	check := New(Config{Client: client})
	err := check(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")
}

// Verify return type matches health.CheckFunc signature.
var _ func(context.Context) error = New(Config{})
