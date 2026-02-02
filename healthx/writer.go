package healthx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ResultWriter writes health check results to an HTTP response.
type ResultWriter interface {
	Write(result *CheckerResult, statusCode int, w http.ResponseWriter, r *http.Request) error
}

// IETFResultWriter implements ResultWriter using IETF health+json format.
// See: https://tools.ietf.org/id/draft-inadarei-api-health-check-06.html
type IETFResultWriter struct {
	showDetails bool // Include per-check details (default false per CONTEXT.md)
	showErrors  bool // Include error messages (default false per CONTEXT.md)
}

// IETFWriterOption configures the IETFResultWriter.
type IETFWriterOption func(*IETFResultWriter)

// NewIETFResultWriter creates a new IETFResultWriter.
// By default, details and error messages are hidden for security.
func NewIETFResultWriter(opts ...IETFWriterOption) *IETFResultWriter {
	w := &IETFResultWriter{
		showDetails: false, // Hidden by default per CONTEXT.md
		showErrors:  false, // Hidden by default per CONTEXT.md
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// WithShowDetails enables per-check details in response.
func WithShowDetails(show bool) IETFWriterOption {
	return func(w *IETFResultWriter) {
		w.showDetails = show
	}
}

// WithShowErrors enables error messages in response.
func WithShowErrors(show bool) IETFWriterOption {
	return func(w *IETFResultWriter) {
		w.showErrors = show
	}
}

// Write implements the ResultWriter interface.
func (rw *IETFResultWriter) Write(
	result *CheckerResult,
	statusCode int,
	w http.ResponseWriter,
	_ *http.Request,
) error {
	resp := rw.toIETFResponse(result)

	w.Header().Set("Content-Type", "application/health+json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return fmt.Errorf("encode health response: %w", err)
	}
	return nil
}

// ietfResponse represents the root of the IETF health check response.
type ietfResponse struct {
	Status string                 `json:"status"`
	Checks map[string][]ietfCheck `json:"checks,omitempty"`
}

// ietfCheck represents a single check in the IETF format.
type ietfCheck struct {
	Status string `json:"status"`
	Time   string `json:"time,omitempty"`
	Output string `json:"output,omitempty"`
}

// toIETFResponse converts a CheckerResult to IETF health+json format.
func (rw *IETFResultWriter) toIETFResponse(result *CheckerResult) ietfResponse {
	resp := ietfResponse{
		Status: mapStatusToIETF(result.Status),
	}

	// Only include checks if showDetails is enabled
	if rw.showDetails && len(result.Details) > 0 {
		resp.Checks = make(map[string][]ietfCheck)

		for name, checkResult := range result.Details {
			check := ietfCheck{
				Status: mapStatusToIETF(checkResult.Status),
				Time:   checkResult.Timestamp.Format(time.RFC3339),
			}

			// Only include error output if showErrors is enabled and there's an error
			if rw.showErrors && checkResult.Error != nil {
				check.Output = checkResult.Error.Error()
			}

			// IETF format uses an array of checks per component
			resp.Checks[name] = []ietfCheck{check}
		}
	}

	return resp
}

// mapStatusToIETF maps internal AvailabilityStatus to IETF status strings.
func mapStatusToIETF(s AvailabilityStatus) string {
	switch s {
	case StatusUp:
		return "pass"
	case StatusDown:
		return "fail"
	case StatusUnknown:
		return "warn"
	default:
		return "warn"
	}
}
