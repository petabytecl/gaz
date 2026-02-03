package gateway

import (
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

// AllowedHeaders is the list of HTTP headers that will be forwarded
// to gRPC as metadata. These headers are commonly used for authentication,
// request tracing, and client identification.
//
//nolint:gochecknoglobals // Package-level configuration for header allowlist.
var AllowedHeaders = []string{
	"authorization",
	"x-request-id",
	"x-correlation-id",
	"x-forwarded-for",
	"x-forwarded-host",
	"accept-language",
}

// allowedHeadersSet is a pre-computed set for O(1) lookup.
//
//nolint:gochecknoglobals // Computed once at init for performance.
var allowedHeadersSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(AllowedHeaders))
	for _, h := range AllowedHeaders {
		set[h] = struct{}{}
	}
	return set
}()

// HeaderMatcher determines which HTTP headers should be forwarded
// to gRPC as metadata. Headers in the AllowedHeaders list are forwarded
// as-is. For other headers, it falls back to the default grpc-gateway behavior.
//
// This function is used with runtime.WithIncomingHeaderMatcher option.
func HeaderMatcher(key string) (string, bool) {
	lowerKey := strings.ToLower(key)
	if _, ok := allowedHeadersSet[lowerKey]; ok {
		return lowerKey, true
	}
	// Fall back to default behavior for standard headers.
	return runtime.DefaultHeaderMatcher(key)
}
