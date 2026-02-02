package internal

import (
	"net/http"
)

// HandlerOption configures the health handler.
type HandlerOption func(*handlerConfig)

// handlerConfig holds the handler configuration.
type handlerConfig struct {
	resultWriter   ResultWriter
	statusCodeUp   int
	statusCodeDown int
}

// NewHandler creates an HTTP handler for health checks.
//
// The handler calls checker.Check() with the request context,
// determines the appropriate status code based on the result,
// and writes the response using the configured ResultWriter.
//
// Default configuration:
//   - ResultWriter: IETFResultWriter (no details, no errors)
//   - StatusCodeUp: 200 OK
//   - StatusCodeDown: 503 Service Unavailable
//
// For liveness-style handlers that should return 200 even on failure,
// set both status codes to 200:
//
//	handler := internal.NewHandler(checker,
//	    internal.WithStatusCodeUp(200),
//	    internal.WithStatusCodeDown(200),
//	)
func NewHandler(checker Checker, opts ...HandlerOption) http.Handler {
	cfg := &handlerConfig{
		resultWriter:   NewIETFResultWriter(),
		statusCodeUp:   http.StatusOK,
		statusCodeDown: http.StatusServiceUnavailable,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := checker.Check(r.Context())

		statusCode := cfg.statusCodeUp
		if result.Status == StatusDown || result.Status == StatusUnknown {
			statusCode = cfg.statusCodeDown
		}

		// Ignore write error - response already started
		_ = cfg.resultWriter.Write(&result, statusCode, w, r)
	})
}

// WithResultWriter sets the response writer (default: IETFResultWriter).
func WithResultWriter(w ResultWriter) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.resultWriter = w
	}
}

// WithStatusCodeUp sets the status code when all checks pass (default: 200).
func WithStatusCodeUp(code int) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.statusCodeUp = code
	}
}

// WithStatusCodeDown sets the status code when any check fails (default: 503).
func WithStatusCodeDown(code int) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.statusCodeDown = code
	}
}
