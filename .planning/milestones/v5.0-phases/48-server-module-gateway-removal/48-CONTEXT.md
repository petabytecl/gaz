# Phase 48: Server Module & Gateway Removal - Context

**Gathered:** 2026-03-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Update `server.NewModule()` to bundle Vanguard as the default server (replacing gateway), remove the `server/gateway/` package entirely, preserve `server/http/` for standalone HTTP-only use cases, and replace the `examples/grpc-gateway/` example with a new Vanguard unified demo.

</domain>

<decisions>
## Implementation Decisions

### Module Bundling API
- `server.NewModule()` = `grpc.NewModule()` + `vanguard.NewModule()` — same composite pattern as today, gateway swapped for Vanguard
- Module automatically sets gRPC `SkipListener=true` so Vanguard handles all connections — user doesn't need to think about it
- Module name stays `"server"` — this IS the server from the user's perspective
- Doc comments on `server/module.go` and `server/doc.go` fully rewritten to reference Vanguard, Connect, gRPC-Web, REST, single-port — all gateway references removed

### Gateway Example Replacement
- Delete `examples/grpc-gateway/` entirely (directory, proto, buf config, generated code, README)
- Create new `examples/vanguard/` with full unified demo: gRPC service + Connect handler + REST via proto annotations + health check, all on one port
- Reuse the existing `hello.proto` from the grpc-gateway example (add `google.api.http` annotations if missing), regenerate with buf for Connect + gRPC stubs
- README updated to show how all protocols work on a single port

### Dependency Cleanup
- Full cleanup — remove all traces of gateway from production code and config:
  - Delete `server/gateway/` package (13 files: config, errors, gateway, handler, headers, module, doc, + tests)
  - Remove `grpc-gateway/v2` from `go.mod` (run `go mod tidy` after removing all imports)
  - Clean `.golangci.yml`: remove gateway from depguard allow lists, ireturn exclusions, any linter path patterns
  - Update `README.md`: remove grpc-gateway example link, add vanguard example link
  - Update `server/doc.go`: rewrite to reference Vanguard architecture
- Audit `grpc-ecosystem/go-grpc-middleware/v2` — check if still used by gRPC interceptor bundles; remove if no longer imported
- Do NOT touch `.planning/` files — they are historical records

### gRPC-Only Mode
- `server.NewModule()` always bundles Vanguard — there is no gRPC-only option in the server module
- Users who want gRPC-only (no Connect, no REST) use `grpc.NewModule()` directly — it already works standalone with `SkipListener=false`
- `server/http` package stays exactly as-is — fully independent, no changes needed (SMOD-03 confirmed)

### Claude's Discretion
- How to auto-set SkipListener=true from the server module (via config override, module option, or direct gRPC config mutation)
- Exact content of the new vanguard example's main.go and service.go
- Whether the vanguard example needs generated Connect stubs or just gRPC stubs with Vanguard handling Connect protocol translation
- go mod tidy handling and verification of transitive dependency removal

</decisions>

<specifics>
## Specific Ideas

- The module composition should feel identical to today: `app.Use(server.NewModule())` gives you a complete server. The internal change from gateway to Vanguard is transparent to users
- The vanguard example should show all four protocols (gRPC, Connect, gRPC-Web, REST) on a single port — this is the key differentiator from the old gateway architecture
- Health endpoints should "just work" in the example without explicit configuration — demonstrates the auto-mount pattern

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `server/module.go`: Current composite module pattern — 3 lines of code, swap gateway.NewModule() for vanguard.NewModule()
- `server/vanguard/module.go`: Already has full NewModule() with 10 providers (config, CORS, OTEL, interceptors, server)
- `examples/grpc-gateway/proto/hello.proto`: Existing proto file to reuse for vanguard example
- `server/grpc/config.go`: Has `SkipListener` field that needs auto-setting from server module

### Established Patterns
- Composite module: `gaz.NewModule("server").Use(grpc.NewModule()).Use(X.NewModule()).Build()`
- Config override from parent module: gRPC module accepts config with SkipListener field
- Auto-discovery: `di.ResolveAll[Interface]` pattern — Vanguard module already uses this for Connect registrars

### Integration Points
- `server/module.go` → imports change from `server/gateway` to `server/vanguard`
- `server/module_test.go` → assertions change from `gateway.Gateway` to `vanguard.Server`
- `.golangci.yml` → depguard and ireturn path patterns need gateway references removed
- `go.mod` → `grpc-gateway/v2` dependency removed after all imports deleted

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 48-server-module-gateway-removal*
*Context gathered: 2026-03-06*
