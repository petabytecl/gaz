package server

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/config"
	"github.com/petabytecl/gaz/di"
	hello "github.com/petabytecl/gaz/examples/vanguard/proto"
	servergrpc "github.com/petabytecl/gaz/server/grpc"
	"github.com/petabytecl/gaz/server/vanguard"
)

func TestNewModule(t *testing.T) {
	// Test with defaults.
	t.Run("defaults", func(t *testing.T) {
		app := gaz.New()

		module := NewModule()
		err := module.Apply(app)
		require.NoError(t, err)

		err = app.Build()
		require.NoError(t, err)

		c := app.Container()

		// Verify servers were registered.
		require.True(t, di.Has[*servergrpc.Server](c))
		require.True(t, di.Has[*vanguard.Server](c))
	})

	// Test gRPC SkipListener is forced true.
	t.Run("grpc skip listener", func(t *testing.T) {
		app := gaz.New()

		module := NewModule()
		err := module.Apply(app)
		require.NoError(t, err)

		err = app.Build()
		require.NoError(t, err)

		c := app.Container()

		cfg, err := di.Resolve[servergrpc.Config](c)
		require.NoError(t, err)
		require.True(t, cfg.SkipListener, "gRPC SkipListener must be true when using server module")
	})

	// Test module name.
	t.Run("module name", func(t *testing.T) {
		module := NewModule()
		require.Equal(t, ModuleName, module.Name())
	})
}

func TestNewModule_ForcesGRPCServiceRegistrationOnly(t *testing.T) {
	app := gaz.New()
	app.Use(NewModule())

	require.NoError(t, app.MergeConfigMap(map[string]any{
		"grpc": map[string]any{
			"port":              43210,
			"reflection":        true,
			"health_enabled":    false,
			"max_recv_msg_size": 1024,
			"max_send_msg_size": 2048,
			"skip_listener":     false,
		},
	}))

	require.NoError(t, app.Build())

	cfg, err := di.Resolve[servergrpc.Config](app.Container())
	require.NoError(t, err)
	require.Equal(t, 43210, cfg.Port)
	require.True(t, cfg.Reflection)
	require.False(t, cfg.HealthEnabled)
	require.Equal(t, 1024, cfg.MaxRecvMsgSize)
	require.Equal(t, 2048, cfg.MaxSendMsgSize)
	require.True(t, cfg.SkipListener)
}

func TestNewModule_RejectsMalformedGRPCOverlay(t *testing.T) {
	app := gaz.New().WithConfig(nil, config.WithBackend(config.NewMapBackend(map[string]any{
		"grpc": "not-a-config-map",
	})))
	app.Use(NewModule())

	err := app.Build()
	require.Error(t, err)
	require.ErrorContains(t, err, "load provider config \"grpc\"")
}

func TestNewModule_BridgesGRPCServicesThroughVanguard(t *testing.T) {
	app := gaz.New()
	app.Use(NewModule())

	cfg := vanguard.DefaultConfig()
	cfg.Port = getFreePort(t)
	cfg.HealthEnabled = false
	cfg.Reflection = false
	cfg.DevMode = true
	require.NoError(t, gaz.For[vanguard.Config](app.Container()).Replace().Instance(cfg))

	require.NoError(t, gaz.For[*bridgeGreeterService](app.Container()).Instance(&bridgeGreeterService{}))

	require.NoError(t, app.Build())
	require.NoError(t, app.Start(context.Background()))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, app.Stop(ctx))
	})

	waitForPort(t, cfg.Port)

	conn, err := googlegrpc.NewClient(
		fmt.Sprintf("localhost:%d", cfg.Port),
		googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})

	client := hello.NewGreeterClient(conn)
	var reply *hello.HelloReply
	var callErr error
	require.Eventually(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		reply, callErr = client.SayHello(ctx, &hello.HelloRequest{Name: "Bridge"})
		return callErr == nil && reply.GetMessage() == "Hello from gRPC, Bridge!"
	}, 2*time.Second, 20*time.Millisecond, "last gRPC error: %v", callErr)
}

type bridgeGreeterService struct {
	hello.UnimplementedGreeterServer
}

func (s *bridgeGreeterService) SayHello(
	_ context.Context,
	req *hello.HelloRequest,
) (*hello.HelloReply, error) {
	return &hello.HelloReply{Message: fmt.Sprintf("Hello from gRPC, %s!", req.GetName())}, nil
}

func (s *bridgeGreeterService) RegisterService(registrar googlegrpc.ServiceRegistrar) {
	hello.RegisterGreeterServer(registrar, s)
}

func getFreePort(t *testing.T) int {
	t.Helper()

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, lis.Close())
	}()

	return lis.Addr().(*net.TCPAddr).Port
}

func waitForPort(t *testing.T, port int) {
	t.Helper()

	addr := fmt.Sprintf("localhost:%d", port)
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 2*time.Second, 10*time.Millisecond)
}
