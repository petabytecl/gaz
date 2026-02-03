package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/petabytecl/gaz/di"
)

// Registrar is implemented by gRPC services that want to expose
// HTTP endpoints via the Gateway. Services call their generated
// RegisterXXXHandler function in this method.
//
// Example implementation:
//
//	type GreeterService struct {
//	    pb.UnimplementedGreeterServer
//	}
//
//	func (s *GreeterService) RegisterGateway(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
//	    return pb.RegisterGreeterHandler(ctx, mux, conn)
//	}
type Registrar interface {
	RegisterGateway(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}

// Gateway is an HTTP-to-gRPC gateway with auto-discovery and CORS support.
// It translates RESTful HTTP/JSON requests into gRPC calls via grpc-gateway.
// Implements di.Starter and di.Stopper for lifecycle integration.
type Gateway struct {
	config      Config
	mux         *runtime.ServeMux
	conn        *grpc.ClientConn
	container   *di.Container
	logger      *slog.Logger
	devMode     bool
	tp          *sdktrace.TracerProvider
	otelEnabled bool
	handler     *DynamicHandler
}

// NewGateway creates a new Gateway with the given configuration.
// The gateway is not started until OnStart is called.
//
// Parameters:
//   - cfg: Gateway configuration (port, gRPC target, CORS)
//   - logger: Logger for request logging and error reporting
//   - container: DI container for service discovery
//   - devMode: If true, expose detailed error messages
//   - tp: Optional TracerProvider for OpenTelemetry instrumentation (may be nil)
func NewGateway(cfg Config, logger *slog.Logger, container *di.Container, devMode bool, tp *sdktrace.TracerProvider) *Gateway {
	if logger == nil {
		logger = slog.Default()
	}
	return &Gateway{
		config:      cfg,
		container:   container,
		logger:      logger,
		devMode:     devMode,
		tp:          tp,
		otelEnabled: tp != nil,
		handler:     NewDynamicHandler(nil),
	}
}

// OnStart initializes the Gateway and registers discovered services.
// It creates a loopback connection to the gRPC server, discovers services
// implementing GatewayRegistrar, and sets up CORS middleware.
// Implements di.Starter.
func (g *Gateway) OnStart(ctx context.Context) error {
	// Determine gRPC target.
	target := g.config.GRPCTarget
	if target == "" {
		target = DefaultGRPCTarget
	}

	// Create loopback connection to gRPC server.
	// Use grpc.NewClient (not deprecated grpc.Dial).
	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("gateway: create grpc client: %w", err)
	}
	g.conn = conn

	// Create ServeMux with options.
	g.mux = runtime.NewServeMux(
		runtime.WithErrorHandler(g.errorHandler),
		runtime.WithIncomingHeaderMatcher(HeaderMatcher),
	)

	// Auto-discover and register services.
	registrars, err := di.ResolveAll[Registrar](g.container)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("gateway: discover registrars: %w", err)
	}

	for _, r := range registrars {
		if regErr := r.RegisterGateway(ctx, g.mux, conn); regErr != nil {
			_ = conn.Close()
			return fmt.Errorf("gateway: register service: %w", regErr)
		}
	}

	// Build CORS handler wrapping mux.
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   g.config.CORS.AllowedOrigins,
		AllowedMethods:   g.config.CORS.AllowedMethods,
		AllowedHeaders:   g.config.CORS.AllowedHeaders,
		ExposedHeaders:   g.config.CORS.ExposedHeaders,
		AllowCredentials: g.config.CORS.AllowCredentials,
		MaxAge:           g.config.CORS.MaxAge,
		Debug:            g.devMode,
	})

	// Build the handler chain
	var h http.Handler = corsHandler.Handler(g.mux)

	// Wrap with OTEL instrumentation if TracerProvider is available.
	// Order: mux -> CORS -> otelhttp (OTEL wraps the outermost layer).
	if g.otelEnabled {
		h = otelhttp.NewHandler(h, "gateway",
			otelhttp.WithFilter(func(r *http.Request) bool {
				// Skip health check endpoints to reduce noise.
				return r.URL.Path != "/health" && r.URL.Path != "/healthz"
			}),
		)
	}

	// Update the dynamic handler atomically.
	g.handler.SetHandler(h)

	g.logger.InfoContext(ctx, "Gateway initialized",
		slog.Int("services", len(registrars)),
		slog.String("grpc_target", target),
		slog.Bool("otel", g.otelEnabled),
	)

	return nil
}

// OnStop gracefully shuts down the Gateway.
// It closes the gRPC client connection.
// Implements di.Stopper.
func (g *Gateway) OnStop(ctx context.Context) error {
	g.logger.InfoContext(ctx, "Gateway stopping")

	if g.conn != nil {
		if err := g.conn.Close(); err != nil {
			return fmt.Errorf("gateway: close grpc connection: %w", err)
		}
	}

	g.logger.InfoContext(ctx, "Gateway stopped")
	return nil
}

// Handler returns the HTTP handler for the Gateway.
// This handler includes CORS middleware and should be used with
// the existing server/http.Server via SetHandler.
func (g *Gateway) Handler() http.Handler {
	return g.handler
}

// errorHandler delegates to the ErrorHandler from errors.go.
func (g *Gateway) errorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	ErrorHandler(g.devMode)(ctx, mux, marshaler, w, r, err)
}
