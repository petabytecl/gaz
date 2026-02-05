# Quick Task 012: Add Builtin gRPC Auth Interceptor

**Status:** Complete
**Date:** 2026-02-05
**Duration:** ~5 minutes

## Summary

Added builtin gRPC authentication interceptor that integrates with go-grpc-middleware/v2/interceptors/auth. The auth interceptor is conditionally registered only when an AuthFunc is present in the DI container, making authentication opt-in.

## Changes

### server/grpc/interceptors.go
- Added `PriorityAuth = 50` constant (after logging, before validation)
- Exported `AuthFunc` type alias for go-grpc-middleware auth.AuthFunc
- Added `AuthBundle` struct implementing InterceptorBundle interface
- Added `NewAuthBundle` constructor

### server/grpc/module.go
- Added `provideAuthBundle` function with conditional registration logic
- Uses `gaz.Has[AuthFunc]` to check if auth is configured
- Updated `NewModule()` to include auth bundle in provider chain
- Updated documentation to include AuthBundle in components list

### server/grpc/interceptors_test.go
- Added `TestAuthBundle_ImplementsInterface` test
- Added `TestPriorityAuth_Ordering` test
- Added `TestProvideAuthBundle_WithAuthFunc` test
- Added `TestProvideAuthBundle_WithoutAuthFunc` test

## Key Decisions

1. **Opt-in Authentication:** Auth interceptor only registered when AuthFunc exists in DI container
2. **Priority Ordering:** Auth priority (50) placed between logging (0) and validation (100)
3. **Type Alias:** Exported AuthFunc as alias to auth.AuthFunc for user convenience
4. **Silent Skip:** No error when AuthFunc not registered - authentication is optional

## Interceptor Chain Order

```
logging (0) -> auth (50) -> validation (100) -> recovery (1000)
```

## Usage Example

```go
// Define your auth function
func myAuthFunc(ctx context.Context) (context.Context, error) {
    token, err := auth.AuthFromMD(ctx, "bearer")
    if err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
    }
    // Validate token and enrich context...
    return ctx, nil
}

// Register in DI to enable auth interceptor
gaz.For[grpc.AuthFunc](c).Instance(myAuthFunc)
```

## Commits

| Hash | Message |
|------|---------|
| 4a4dc0f | feat(012): add AuthBundle to grpc interceptors |
| 136cd8a | feat(012): add conditional auth bundle provider |
| 0e158ee | test(012): add AuthBundle tests and fix lint issues |

## Verification

- [x] `go build ./server/grpc/...` - compiles
- [x] `go test -race ./server/grpc/...` - all tests pass
- [x] `make lint` - no linting issues
- [x] `make cover` - 90.2% coverage maintained
