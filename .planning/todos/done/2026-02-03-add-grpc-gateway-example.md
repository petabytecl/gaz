---
created: 2026-02-03T00:00
title: Add gRPC-Gateway example
area: docs
files:
  - examples/grpc-gateway
---

## Problem

Users need a working example demonstrating how to integrate gRPC and Gateway servers using `gaz`. Implementing a full proto build system for an example is unnecessary overhead and complexity.

## Solution

Create a new example in `examples/grpc-gateway` using a minimal `hello.proto` definition. 
Generate the gRPC and Gateway code artifacts locally once and commit them to the repository.
This allows the example to be runnable without `protoc` installation while keeping the codebase clean and under our control.

The example should demonstrate:
- `server.NewModule` / `NewModuleWithFlags`
- `gaz` dependency injection wiring
- Dual port architecture (gRPC + Gateway)
