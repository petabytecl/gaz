package health

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/alexliesenfeld/health"
)

// IETFResultWriter implements health.ResultWriter using the IETF JSON format.
// See: https://tools.ietf.org/id/draft-inadarei-api-health-check-06.html
type IETFResultWriter struct{}

// NewIETFResultWriter creates a new IETFResultWriter.
func NewIETFResultWriter() *IETFResultWriter {
	return &IETFResultWriter{}
}

// Write implements the health.ResultWriter interface.
func (rw *IETFResultWriter) Write(result *health.CheckerResult, statusCode int, w http.ResponseWriter, r *http.Request) error {
	resp := toIETFResponse(result)

	w.Header().Set("Content-Type", "application/health+json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(resp)
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

func toIETFResponse(result *health.CheckerResult) ietfResponse {
	resp := ietfResponse{
		Status: mapStatus(result.Status),
		Checks: make(map[string][]ietfCheck),
	}

	for name, checkResult := range result.Details {
		check := ietfCheck{
			Status: mapStatus(checkResult.Status),
			Time:   checkResult.Timestamp.Format(time.RFC3339),
		}

		if checkResult.Error != nil {
			check.Output = checkResult.Error.Error()
		}

		// IETF format uses an array of checks per component.
		resp.Checks[name] = []ietfCheck{check}
	}

	return resp
}

func mapStatus(s health.AvailabilityStatus) string {
	switch s {
	case health.StatusUp:
		return "pass"
	case health.StatusDown:
		return "fail"
	case health.StatusUnknown:
		return "warn"
	default:
		return "warn"
	}
}
