# Security Guide

This guide covers security best practices for gaz applications.

## Input Validation

### Service and Module Names

All service, module, and worker names are validated to prevent injection attacks. Names must be alphanumeric with hyphens or underscores only.

**Valid names:**
- `database`
- `user-service`
- `cache_manager`
- `api_v2`

**Invalid names:**
- `service/../other` (path traversal attempt)
- `service<script>` (special characters)
- `service name` (spaces)

### Configuration Keys

Configuration keys are validated to prevent path traversal and injection:

**Valid keys:**
- `database.host`
- `redis-config.port`
- `api.timeout`

**Invalid keys:**
- `../etc/passwd` (path traversal)
- `key<script>` (special characters)
- `key name` (spaces)

### Validation Errors

Invalid names/keys return `ErrInvalidName`:

```go
err := di.For[*Service](c).Named("invalid-name!")
if errors.Is(err, di.ErrInvalidName) {
    // Handle validation error
}
```

## Resource Limits

The framework enforces resource limits to prevent DoS attacks:

### Container Limits

```go
opts := di.DefaultContainerOptions()
opts.MaxServices = 500 // Limit service registrations
c := di.NewWithOptions(opts)
```

### Worker Limits

```go
opts := worker.DefaultWorkerOptions()
opts.MaxWorkers = 50 // Limit worker registrations
mgr := worker.NewManagerWithOptions(logger, opts)
```

### EventBus Limits

```go
bus := eventbus.NewWithOptions(logger, 500) // Max 500 subscriptions
```

### Health Check Limits

```go
mgr := health.NewManagerWithOptions(50) // Max 50 health checks
```

### Default Limits

- **MaxServices**: 1000
- **MaxWorkers**: 100
- **MaxSubscriptions**: 1000
- **MaxHealthChecks**: 100

These defaults are reasonable for most applications but can be adjusted based on your needs.

## Error Handling Security

### Information Disclosure

Error messages are designed to avoid exposing sensitive information:

- **No sensitive data** - Errors don't include connection strings, passwords, or internal paths
- **No stack traces** - Stack traces only appear in panic recovery logs, not normal errors
- **Structured errors** - Use sentinel errors for programmatic handling without exposing details

### Error Message Format

All error messages follow a consistent format:

```
action: underlying error
```

- Lowercase action description
- No trailing punctuation
- Wrapped underlying errors for context

Example:
```go
// Good
return wrapErr("register service", err)

// Avoid
return fmt.Errorf("Failed to register service: %v", err) // Exposes details
```

## Dependency Injection Security

### Type Safety

The framework uses generics for compile-time type safety:

```go
// Type-safe resolution
db, err := gaz.Resolve[*Database](c)
```

This prevents:
- Type confusion attacks
- Runtime type assertion failures
- Injection of wrong types

### Cycle Detection

Circular dependencies are detected and prevented:

```go
// This will fail with ErrCycle
di.For[*A](c).Provider(func(c *di.Container) (*A, error) {
    b, _ := di.Resolve[*B](c)
    return &A{b: b}, nil
})
di.For[*B](c).Provider(func(c *di.Container) (*B, error) {
    a, _ := di.Resolve[*A](c) // Cycle detected!
    return &B{a: a}, nil
})
```

### No Code Generation

The framework doesn't use code generation, reducing attack surface:
- No generated code to audit
- No build-time injection points
- Explicit provider functions only

## Configuration Security

### Environment Variables

Environment variables override config file values. Be careful with:
- **Secrets** - Never log environment variable values
- **Validation** - Always validate config values
- **Defaults** - Use secure defaults

### Config File Security

- **File permissions** - Restrict config file access (chmod 600)
- **Path validation** - Config file paths are validated
- **No remote loading** - Config files are loaded from local filesystem only

### Struct Validation

Use struct tags for validation:

```go
type Config struct {
    Host     string `validate:"required,hostname"`
    Port     int    `validate:"required,min=1,max=65535"`
    Password string `validate:"required,min=8"`
}
```

## Production Deployment Checklist

- [ ] Set resource limits appropriate for your workload
- [ ] Validate all user-provided names and keys
- [ ] Use secure defaults for configuration
- [ ] Restrict config file permissions
- [ ] Don't log sensitive configuration values
- [ ] Enable health checks for resource monitoring
- [ ] Monitor goroutine and memory usage
- [ ] Use structured logging (no sensitive data in logs)
- [ ] Review error messages for information disclosure
- [ ] Test resource limit enforcement
- [ ] Validate all input at boundaries

## Common Pitfalls

### Pitfall 1: Exposing Sensitive Data in Errors

**Bad:**
```go
return fmt.Errorf("database connection failed: %s", connStr)
```

**Good:**
```go
return wrapErr("connect to database", err) // err doesn't include connStr
```

### Pitfall 2: No Resource Limits

**Bad:**
```go
// No limits - vulnerable to DoS
c := di.New()
for i := 0; i < 100000; i++ {
    di.For[*Service](c).Named(fmt.Sprintf("svc-%d", i))
}
```

**Good:**
```go
opts := di.DefaultContainerOptions()
opts.MaxServices = 1000
c := di.NewWithOptions(opts) // Enforces limit
```

### Pitfall 3: Invalid Names

**Bad:**
```go
di.For[*Service](c).Named("service/../other") // Path traversal attempt
```

**Good:**
```go
di.For[*Service](c).Named("service-other") // Validated and safe
```

## Security Best Practices

1. **Validate Early** - Validate all input at registration time
2. **Limit Resources** - Set appropriate limits for your use case
3. **Monitor Resources** - Use health checks to detect exhaustion
4. **Secure Defaults** - Don't expose sensitive data by default
5. **Error Handling** - Don't leak sensitive information in errors
6. **Logging** - Use structured logging, avoid sensitive data
7. **Configuration** - Restrict file permissions, validate values
8. **Dependencies** - Keep dependencies up to date

## Reporting Security Issues

If you discover a security vulnerability, please report it responsibly:
- Do not open public issues
- Contact maintainers privately
- Provide detailed reproduction steps
- Allow time for fix before disclosure
