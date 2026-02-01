# Gemini Context: gaz

## Project Overview
`gaz` is a simple, type-safe dependency injection library for Go applications with built-in lifecycle management. It avoids code generation and reflection magic, relying on Go generics for type safety. It provides a robust foundation for building modular applications with integrated support for configuration, health checks, background workers, cron jobs, and an event bus.

## Key Technologies
*   **Language:** Go (1.25+)
*   **Core Dependencies:**
    *   `spf13/cobra` (CLI integration)
    *   `spf13/viper` (Configuration management)
    *   `alexliesenfeld/health` (Health check handling)
    *   `go-playground/validator` (Struct validation)
    *   `robfig/cron` (Job scheduling)

## Build & Run
The project uses a `Makefile` to orchestrate development workflows.

*   **Run Tests:** `make test` (Runs `go test -race ./...`)
*   **Check Coverage:** `make cover`
    *   **Note:** Enforces a strict **90% coverage threshold**. Fails if coverage drops below this.
*   **Format Code:** `make fmt` (Applies `gofmt -w .` and `goimports -w .`)
*   **Check Formatting:** `make fmt-check`
*   **Lint Code:** `make lint` (Runs `golangci-lint`)
*   **Clean:** `make clean` (Removes generated artifacts like `coverage.out`)

## Development Conventions
Strict adherence to `STYLE.md` is required.

### Naming Standards
*   **Packages:** strictly lowercase, short, no underscores (e.g., `package health`, not `package health_check`).
*   **Types:** PascalCase for exported, camelCase for unexported.
*   **Interfaces:**
    *   Single-method: `-er` suffix (e.g., `Starter`).
    *   Multi-method: Descriptive noun (e.g., `Backend`).

### Constructor Patterns
*   **`New()`**: Use when the package exports a single primary type.
*   **`NewX()`**: Use when the package exports multiple types to avoid ambiguity.
*   **`NewXWithY()`**: Use for variant constructors with alternative setups.
*   **Builder Pattern**: Preferred for types with many optional configurations.

### Error Handling
*   **Sentinel Variables:** MUST use `Err` prefix (e.g., `ErrNotFound`).
*   **Message Format:** `"pkg: description"` (lowercase, no trailing punctuation).
*   **Wrapping:** MUST use `fmt.Errorf("context: %w", err)` to preserve error chains.

### Module Design
*   **Factory Signature:** MUST be `func Module(c *gaz.Container) error`.
*   **Error Propagation:** Registration errors MUST be wrapped with context (e.g., `"register manager: %w"`).

## Project Structure
*   **`gaz` (root):** Core `App` and `Container` types, high-level API entry points.
*   **`config/`:** Configuration loading strategies (File, Env) and validation logic.
*   **`di/`:** The heart of the dependency injection engine (Lifecycle, Registration, Resolution).
*   **`health/`:** Health check manager, shutdown coordination, and HTTP handlers.
*   **`worker/`:** Background worker supervision, including backoff strategies and circuit breakers.
*   **`cron/`:** Integration with `robfig/cron` for scheduled tasks.
*   **`eventbus/`:** Simple in-memory event bus for decoupling components.
*   **`logger/`:** Logging interfaces and middleware.
*   **`examples/`:** Reference implementations showing various features (CLI, HTTP, Microservices).
