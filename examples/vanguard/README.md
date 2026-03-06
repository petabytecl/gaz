# Vanguard Example

This example demonstrates the **unified server module** (`server.NewModule()`) which serves
all four protocols on a single port:

| Protocol  | How to test |
|-----------|-------------|
| **REST**  | `curl -X POST http://localhost:8080/v1/example/echo -H "Content-Type: application/json" -d '{"name": "World"}'` |
| **Connect** | `curl -X POST http://localhost:8080/hello.Greeter/SayHello -H "Content-Type: application/json" -d '{"name": "World"}'` |
| **gRPC**  | `grpcurl -plaintext -d '{"name": "World"}' localhost:8080 hello.Greeter/SayHello` |
| **gRPC-Web** | Use a gRPC-Web client library (e.g., `@connectrpc/connect-web`) |

## Running

```bash
go run .
```

The server starts on port **8080** by default. Use `--server-port` to change it.

## How It Works

1. **`server.NewModule()`** bundles the gRPC module and the Vanguard module.
   gRPC registers services and interceptors but skips its own listener (`SkipListener=true`).
   Vanguard handles all inbound connections on a single h2c port.

2. **`GreeterService`** implements two interfaces for auto-discovery:
   - `server/grpc.Registrar` — registers with the gRPC server
   - `server/connect.Registrar` — registers with the Connect/Vanguard handler

3. **Proto HTTP annotations** (in `proto/hello.proto`) enable REST transcoding:
   ```protobuf
   rpc SayHello (HelloRequest) returns (HelloReply) {
     option (google.api.http) = {
       post: "/v1/example/echo"
       body: "*"
     };
   }
   ```

## Project Structure

```
examples/vanguard/
├── main.go                          # App entry point
├── service.go                       # GreeterService + Connect adapter
├── proto/
│   ├── hello.proto                  # Protobuf definition with HTTP annotations
│   ├── hello.pb.go                  # Generated protobuf types
│   ├── hello_grpc.pb.go            # Generated gRPC stubs
│   └── helloconnect/
│       └── hello.connect.go        # Generated Connect stubs
├── buf.yaml                         # Buf module config
├── buf.gen.yaml                     # Buf code generation config
└── README.md                        # This file
```

## Regenerating Proto Code

```bash
buf dep update
buf generate
```
