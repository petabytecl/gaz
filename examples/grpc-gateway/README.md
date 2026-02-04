# gRPC-Gateway Example

This example demonstrates a combined gRPC server with HTTP/REST gateway using the `gaz` framework. It showcases:

- Unified server module bundling gRPC + Gateway with proper lifecycle management
- Auto-discovery of service registrations via DI
- Protocol Buffers with gRPC-Gateway annotations for HTTP/REST endpoints
- Health checks for both gRPC and HTTP
- OpenTelemetry integration ready

## Directory Structure

```
grpc-gateway/
├── main.go              # Application entry point with Cobra CLI
├── service.go           # GreeterService implementation
├── buf.yaml             # Buf module configuration
├── buf.gen.yaml         # Buf code generation configuration
├── buf.lock             # Buf dependency lock file
└── proto/
    ├── hello.proto      # Protocol Buffer service definition
    ├── hello.pb.go      # Generated Go protobuf types
    ├── hello_grpc.pb.go # Generated gRPC server/client code
    └── hello.pb.gw.go   # Generated HTTP-to-gRPC gateway code
```

## Service Definition

The example defines a simple `Greeter` service:

| Component    | Description                         |
|--------------|-------------------------------------|
| Service      | `Greeter`                           |
| RPC Method   | `SayHello`                          |
| Request      | `HelloRequest { string name }`      |
| Response     | `HelloReply { string message }`     |
| HTTP Mapping | `POST /v1/example/echo` (body: `*`) |

## Running the Server

### Basic Usage

```bash
# Run with default ports (gRPC: 50051, HTTP: 8080)
go run . serve

# Run with custom ports
go run . serve --grpc-port 9090 --gateway-port 8080

# Run with development mode (verbose errors, wide-open CORS)
go run . serve --grpc-dev-mode --gateway-dev-mode
```

### Available Flags

| Flag                     | Default                   | Description                      |
|--------------------------|---------------------------|----------------------------------|
| `--grpc-port`            | `50051`                   | gRPC server port                 |
| `--grpc-reflection`      | `true`                    | Enable gRPC reflection           |
| `--grpc-dev-mode`        | `false`                   | Enable verbose error messages    |
| `--grpc-health-enabled`  | `true`                    | Enable gRPC health check service |
| `--grpc-health-interval` | `5s`                      | Interval for health status sync  |
| `--gateway-port`         | `8080`                    | Gateway HTTP port                |
| `--gateway-grpc-target`  | `localhost:<grpc-port>`   | gRPC server target               |
| `--gateway-dev-mode`     | `false`                   | Enable wide-open CORS settings   |

## Testing the Service

### Using curl (HTTP/REST Gateway)

```bash
# POST request with JSON body
curl -X POST http://localhost:8080/v1/example/echo \
  -H "Content-Type: application/json" \
  -d '{"name": "World"}'

# Expected response:
# {"message":"Hello, World!"}
```

### Using grpcurl (gRPC)

```bash
# Install grpcurl if not already installed
# macOS: brew install grpcurl
# Linux: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List available services (reflection enabled by default)
grpcurl -plaintext localhost:50051 list

# Describe the Greeter service
grpcurl -plaintext localhost:50051 describe hello.Greeter

# Call SayHello method
grpcurl -plaintext \
  -d '{"name": "Developer"}' \
  localhost:50051 hello.Greeter/SayHello

# Expected response:
# {
#   "message": "Hello, Developer!"
# }
```

### Using grpc-client-cli

```bash
# Install grpc-client-cli
# go install github.com/vadimi/grpc-client-cli/cmd/grpc-client-cli@latest

# Interactive mode
grpc-client-cli localhost:50051

# Direct call
echo '{"name": "Gaz"}' | grpc-client-cli -service hello.Greeter -method SayHello localhost:50051
```

### Health Checks

```bash
# HTTP health endpoint
curl http://localhost:8080/health

# gRPC health check (using grpcurl)
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check

# Check specific service health
grpcurl -plaintext \
  -d '{"service": "hello.Greeter"}' \
  localhost:50051 grpc.health.v1.Health/Check
```

## Regenerating Proto Files

If you modify `proto/hello.proto`, regenerate the Go code using buf:

```bash
# Install buf if not already installed
# https://buf.build/docs/installation

# Generate code
buf generate

# Or with verbose output
buf generate -v
```

## Code Highlights

### Service Auto-Registration

The `GreeterService` implements two interfaces for automatic discovery:

```go
// For gRPC registration
func (g *GreeterService) RegisterService(s grpc.ServiceRegistrar) {
    hello.RegisterGreeterServer(s, g)
}

// For HTTP Gateway registration
func (g *GreeterService) RegisterGateway(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
    return hello.RegisterGreeterHandler(ctx, mux, conn)
}
```

### Module Composition

The application uses `server.NewModule()` which bundles gRPC and Gateway with correct startup order:

```go
app, err := gaz.New(
    gaz.WithCobra(serveCmd),
    gaz.WithModules(
        loggermod.New(),
        configmod.New(),
        server.NewModule(),  // gRPC + Gateway bundled
    ),
)
```
