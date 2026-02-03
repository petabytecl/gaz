package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/status"
)

// ProblemDetails represents an RFC 7807 Problem Details response.
// This format provides a standardized way to communicate errors in HTTP APIs.
type ProblemDetails struct {
	// Type is a URI reference that identifies the problem type.
	// When dereferenced, it provides human-readable documentation.
	Type string `json:"type"`

	// Title is a short, human-readable summary of the problem type.
	Title string `json:"title"`

	// Status is the HTTP status code for this occurrence.
	Status int `json:"status"`

	// Detail is a human-readable explanation specific to this occurrence.
	// Omitted in production mode to avoid information disclosure.
	Detail string `json:"detail,omitempty"`

	// Instance is a URI reference that identifies the specific occurrence.
	// Used for correlation (typically X-Request-ID).
	Instance string `json:"instance,omitempty"`

	// Code is the gRPC status code name (dev mode only).
	Code string `json:"code,omitempty"`
}

// ErrorHandler returns a runtime.ErrorHandlerFunc that creates RFC 7807
// Problem Details responses. In dev mode, detailed error information is
// included. In production mode, generic messages are used to prevent
// information disclosure.
func ErrorHandler(devMode bool) runtime.ErrorHandlerFunc {
	return func(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
		s := status.Convert(err)
		httpStatus := runtime.HTTPStatusFromCode(s.Code())

		problem := ProblemDetails{
			Type:     fmt.Sprintf("https://grpc.io/docs/guides/status-codes/#%s", strings.ToLower(s.Code().String())),
			Title:    s.Code().String(),
			Status:   httpStatus,
			Instance: r.Header.Get("X-Request-ID"),
		}

		if devMode {
			problem.Detail = s.Message()
			problem.Code = s.Code().String()
		} else {
			// Use generic message for production to avoid information disclosure.
			problem.Detail = http.StatusText(httpStatus)
		}

		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(httpStatus)
		_ = json.NewEncoder(w).Encode(problem)
	}
}
