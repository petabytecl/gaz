# Technology Stack

**Project:** Security & Hardening v1.1
**Researched:** Mon Jan 26 2026

## Recommended Stack

### Validation Library
| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `go-playground/validator` | v10 | Struct-based validation | The de-facto standard for struct validation in Go. Integrates seamlessly with struct tags used by `koanf`. |

### Standard Library Components
| Component | Purpose | Usage |
|-----------|---------|-------|
| `context` | Timeout management | Use `context.WithTimeout` to enforce shutdown limits. |
| `os/signal` | Signal handling | Intercept `SIGTERM`/`SIGINT` to trigger the guarded shutdown. |

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Validation | `go-playground/validator` | `ozzo-validation` | `ozzo` is more verbose and procedural. `validator` uses struct tags which aligns better with the declarative nature of `koanf` struct mapping. |
| Config | Keep `koanf` | Viper | No need to replace the existing config loader; `koanf` is lightweight and sufficient. |

## Integration

```bash
# Add validator
go get github.com/go-playground/validator/v10
```
