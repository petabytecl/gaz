---
quick: 005
type: review
description: Review all v4.1 milestone packages for implementation consistency and clean standards
autonomous: true
files_modified:
  - server/otel/config.go
  - server/grpc/module.go
  - server/gateway/module.go
  - server/otel/module.go
  - server/http/doc.go
  - server/otel/config_test.go
---

<objective>
Align v4.1 milestone packages (grpc, http, gateway, otel, health/grpc) to consistent implementation standards.

Purpose: Ensure all new server packages follow the same patterns established in the codebase for Config, Module, error handling, and logging.

Output: Consistent implementations across all v4.1 packages with verified test passing.
</objective>

<context>
@.planning/STATE.md
@AGENTS.md
</context>

<review_findings>

## Inconsistencies Found

### 1. Config Pattern (SetDefaults, Validate)

| Package | DefaultConfig() | SetDefaults() | Validate() | Struct Tags |
|---------|----------------|---------------|------------|-------------|
| grpc | Yes | Yes | Yes | Yes (json,yaml,mapstructure) |
| http | Yes | Yes | Yes | Yes (json,yaml,mapstructure) |
| gateway | Yes | Yes | Yes | Yes (json,yaml,mapstructure) |
| otel | Yes | **NO** | **NO** | **NO** |

**Issue:** `server/otel/config.go` is missing:
- Struct tags on Config fields
- `SetDefaults()` method
- `Validate()` method

### 2. Logger Resolution (slog.Default() fallback)

| Package | Logger Fallback | Pattern |
|---------|-----------------|---------|
| grpc | No - returns error | Strict |
| http | Yes - slog.Default() | Lenient |
| gateway | No - returns error | Strict |
| otel | No - returns error | Strict |
| health/grpc | Yes - slog.Default() | Lenient |

**Issue:** Inconsistent logger handling. grpc and gateway require logger registration while http and health allow fallback.

**Recommendation:** Standardize on slog.Default() fallback for robustness (matching http/health pattern). Logger module should be optional.

### 3. doc.go Quality

| Package | Lines | Sections | Quality |
|---------|-------|----------|---------|
| grpc | 68 | Overview, Quick Start, Service Reg, Interceptors, Reflection, Config | Comprehensive |
| http | 42 | Basic Usage, Custom Config, Handler, Lifecycle | Sparse |
| gateway | 69 | Overview, Auto-Discovery, Usage, CORS, Errors | Comprehensive |
| otel | 43 | Overview, Auto-Enable, Config, Usage, Instrumentation, Graceful | Good |
| server | 48 | Overview, Usage, Options, Subpackages, Lifecycle | Good |

**Issue:** `server/http/doc.go` lacks:
- Configuration section with YAML example
- Timeout explanation section
- Security section (slow loris prevention)

</review_findings>

<tasks>

<task type="auto">
  <name>Task 1: Align otel/config.go with established pattern</name>
  <files>server/otel/config.go, server/otel/config_test.go</files>
  <action>
    1. Add struct tags to Config fields (json, yaml, mapstructure) matching grpc/http/gateway pattern:
       - Endpoint: `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
       - ServiceName: `json:"service_name" yaml:"service_name" mapstructure:"service_name"`
       - SampleRatio: `json:"sample_ratio" yaml:"sample_ratio" mapstructure:"sample_ratio"`
       - Insecure: `json:"insecure" yaml:"insecure" mapstructure:"insecure"`

    2. Add SetDefaults() method matching grpc/http/gateway pattern:
       ```go
       func (c *Config) SetDefaults() {
           if c.ServiceName == "" {
               c.ServiceName = "gaz"
           }
           if c.SampleRatio <= 0 {
               c.SampleRatio = DefaultSampleRatio
           }
           // Insecure defaults to false (Go zero value is correct)
           // Endpoint empty means disabled (intentional, no default)
       }
       ```

    3. Add Validate() method matching other packages:
       ```go
       func (c *Config) Validate() error {
           if c.SampleRatio < 0 || c.SampleRatio > 1.0 {
               return fmt.Errorf("otel: invalid sample_ratio %f: must be between 0.0 and 1.0", c.SampleRatio)
           }
           if c.Endpoint != "" && c.ServiceName == "" {
               return fmt.Errorf("otel: service_name required when endpoint is set")
           }
           return nil
       }
       ```

    4. Add tests for SetDefaults() and Validate() in config_test.go.
  </action>
  <verify>
    - `go build ./server/otel/...` succeeds
    - `go test -race ./server/otel/...` passes
    - Config struct has all tags
    - SetDefaults() and Validate() methods exist
  </verify>
  <done>otel/config.go matches grpc/http/gateway Config pattern with tags, SetDefaults(), Validate()</done>
</task>

<task type="auto">
  <name>Task 2: Standardize logger fallback pattern</name>
  <files>server/grpc/module.go, server/gateway/module.go, server/otel/module.go</files>
  <action>
    Standardize all modules to use slog.Default() fallback pattern (matching http and health):

    1. In server/grpc/module.go Module() function, change lines 110-113:
       FROM:
       ```go
       logger, err := di.Resolve[*slog.Logger](c)
       if err != nil {
           return nil, fmt.Errorf("resolve logger: %w", err)
       }
       ```
       TO:
       ```go
       logger := slog.Default()
       if resolved, resolveErr := di.Resolve[*slog.Logger](c); resolveErr == nil {
           logger = resolved
       }
       ```

    2. In server/gateway/module.go Module() function, change lines 194-197:
       FROM:
       ```go
       logger, err := di.Resolve[*slog.Logger](c)
       if err != nil {
           return nil, fmt.Errorf("resolve logger: %w", err)
       }
       ```
       TO:
       ```go
       logger := slog.Default()
       if resolved, resolveErr := di.Resolve[*slog.Logger](c); resolveErr == nil {
           logger = resolved
       }
       ```

    3. In server/otel/module.go registerTracerProvider() function, change lines 141-144:
       FROM:
       ```go
       logger, err := di.Resolve[*slog.Logger](c)
       if err != nil {
           return nil, fmt.Errorf("resolve logger: %w\", err)
       }
       ```
       TO:
       ```go
       logger := slog.Default()
       if resolved, resolveErr := di.Resolve[*slog.Logger](c); resolveErr == nil {
           logger = resolved
       }
       ```

    This makes logger module optional across all v4.1 packages, matching http and health behavior.
  </action>
  <verify>
    - `go build ./server/...` succeeds
    - `go test -race ./server/...` passes
    - All module.go files use consistent logger fallback pattern
  </verify>
  <done>All server modules use slog.Default() fallback, logger module is optional</done>
</task>

<task type="auto">
  <name>Task 3: Enhance http/doc.go to match quality of other packages</name>
  <files>server/http/doc.go</files>
  <action>
    Expand http/doc.go to match the quality of grpc/gateway docs (~60-70 lines):

    1. Add Configuration section with YAML example:
       ```
       # Configuration

       Configuration can be provided via config file or module options:

           servers:
             http:
               port: 8080
               read_timeout: 10s
               write_timeout: 30s
               idle_timeout: 120s
               read_header_timeout: 5s
       ```

    2. Add Timeout Rationale section explaining each timeout:
       - ReadTimeout: Prevents clients from keeping connections open indefinitely
       - WriteTimeout: Ensures server doesn't hang on slow clients
       - IdleTimeout: Manages keep-alive connection lifecycle
       - ReadHeaderTimeout: Prevents slow loris attacks (5s is recommended)

    3. Add Security Considerations section:
       - Mention slow loris protection via ReadHeaderTimeout
       - Reference to RFC/research if appropriate

    4. Reorder sections to match grpc doc.go structure:
       Overview -> Quick Start -> Configuration -> Timeouts -> Security -> Lifecycle
  </action>
  <verify>
    - `go doc ./server/http` shows enhanced documentation
    - Doc includes Configuration, Timeout, and Security sections
    - Line count increased to ~60-70 lines
  </verify>
  <done>http/doc.go matches quality and structure of grpc/gateway/otel docs</done>
</task>

</tasks>

<verification>
After all tasks:
1. `make lint` passes
2. `make test` passes
3. All v4.1 packages follow consistent patterns:
   - Config: DefaultConfig(), SetDefaults(), Validate(), struct tags
   - Module: slog.Default() logger fallback
   - doc.go: Comprehensive with examples
</verification>

<success_criteria>
- otel/config.go has SetDefaults(), Validate(), and struct tags
- All server/*/module.go use slog.Default() fallback pattern
- http/doc.go enhanced to ~60-70 lines with Configuration, Timeouts, Security sections
- All tests pass, linter clean
</success_criteria>

<output>
After completion, update STATE.md Quick Tasks table and commit changes.
</output>
