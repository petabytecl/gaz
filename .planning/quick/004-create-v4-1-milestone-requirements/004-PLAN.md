---
phase: quick
plan: 004
type: execute
wave: 1
depends_on: []
files_modified: [specs/v4.1-requirements.md]
autonomous: true
must_haves:
  truths:
    - "Detailed requirements document for v4.1 exists"
  artifacts:
    - path: "specs/v4.1-requirements.md"
      provides: "Milestone specifications"
---

<objective>
Capture requirements for v4.1 milestone (HTTP/gRPC/Gateway support) in a structured specification document.

Purpose: Provide clear blueprint for the next /gsd-new-milestone run.
Output: specs/v4.1-requirements.md
</objective>

<tasks>

<task type="auto">
  <name>Task 1: Create v4.1 Requirements Specification</name>
  <files>specs/v4.1-requirements.md</files>
  <action>
    Create a new directory `specs` if it doesn't exist.
    Create `specs/v4.1-requirements.md` with the following structure:

    1.  **Milestone Overview**: v4.1 Server & Transport Layer
    2.  **Core Features**:
        *   **HTTP Server**: Standard lib `net/http` using Go 1.22+ `http.ServeMux`.
        *   **gRPC Server**: `google.golang.org/grpc`.
        *   **gRPC-Gateway**:
            *   Must run on a separate port from gRPC server.
            *   Dynamic service registration pattern (adapt `_tmp_trust/grpcx/gateway.go` concept).
            *   `Service` interface for binding handlers.
            *   Middleware: CORS, OTEL (OpenTelemetry), TLS support.
    3.  **Infrastructure**:
        *   **Health Checks**: Add `jackc/pgx` support to `health/checks`.
    4.  **Architecture Guidelines**:
        *   **Port Separation**: Explicit separation of HTTP and gRPC ports.
        *   **DI Integration**: Adapt `uber/fx` lifecycle patterns to `gaz` (Starter/Stopper interfaces).
        *   **Module Design**: How these servers expose themselves to the `gaz` container.
  </action>
  <verify>
    ls -l specs/v4.1-requirements.md
  </verify>
  <done>Requirements document exists with all specified sections.</done>
</task>

</tasks>

<success_criteria>
- [ ] specs/v4.1-requirements.md created
- [ ] Includes HTTP, gRPC, and Gateway requirements
- [ ] Includes Health Check updates
- [ ] Includes Architecture/Integration patterns
</success_criteria>
