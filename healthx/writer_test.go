package healthx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewIETFResultWriter(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		w := NewIETFResultWriter()
		if w.showDetails {
			t.Error("showDetails should be false by default")
		}
		if w.showErrors {
			t.Error("showErrors should be false by default")
		}
	})

	t.Run("with options", func(t *testing.T) {
		w := NewIETFResultWriter(
			WithShowDetails(true),
			WithShowErrors(true),
		)
		if !w.showDetails {
			t.Error("showDetails should be true")
		}
		if !w.showErrors {
			t.Error("showErrors should be true")
		}
	})
}

func TestIETFResultWriter_Write(t *testing.T) {
	fixedTime := time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		result         *CheckerResult
		statusCode     int
		showDetails    bool
		showErrors     bool
		wantStatus     string
		wantHTTPStatus int
		wantChecks     bool
	}{
		{
			name: "status up without details",
			result: &CheckerResult{
				Status: StatusUp,
				Details: map[string]CheckResult{
					"db": {Status: StatusUp, Timestamp: fixedTime},
				},
			},
			statusCode:     http.StatusOK,
			showDetails:    false,
			showErrors:     false,
			wantStatus:     "pass",
			wantHTTPStatus: http.StatusOK,
			wantChecks:     false,
		},
		{
			name: "status down without details",
			result: &CheckerResult{
				Status: StatusDown,
				Details: map[string]CheckResult{
					"db": {Status: StatusDown, Timestamp: fixedTime, Error: errors.New("connection failed")},
				},
			},
			statusCode:     http.StatusServiceUnavailable,
			showDetails:    false,
			showErrors:     false,
			wantStatus:     "fail",
			wantHTTPStatus: http.StatusServiceUnavailable,
			wantChecks:     false,
		},
		{
			name: "status unknown maps to warn",
			result: &CheckerResult{
				Status:  StatusUnknown,
				Details: map[string]CheckResult{},
			},
			statusCode:     http.StatusServiceUnavailable,
			showDetails:    false,
			showErrors:     false,
			wantStatus:     "warn",
			wantHTTPStatus: http.StatusServiceUnavailable,
			wantChecks:     false,
		},
		{
			name: "with details enabled",
			result: &CheckerResult{
				Status: StatusUp,
				Details: map[string]CheckResult{
					"db":    {Status: StatusUp, Timestamp: fixedTime},
					"cache": {Status: StatusUp, Timestamp: fixedTime},
				},
			},
			statusCode:     http.StatusOK,
			showDetails:    true,
			showErrors:     false,
			wantStatus:     "pass",
			wantHTTPStatus: http.StatusOK,
			wantChecks:     true,
		},
		{
			name: "with details and errors enabled",
			result: &CheckerResult{
				Status: StatusDown,
				Details: map[string]CheckResult{
					"db": {Status: StatusDown, Timestamp: fixedTime, Error: errors.New("connection failed")},
				},
			},
			statusCode:     http.StatusServiceUnavailable,
			showDetails:    true,
			showErrors:     true,
			wantStatus:     "fail",
			wantHTTPStatus: http.StatusServiceUnavailable,
			wantChecks:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewIETFResultWriter(
				WithShowDetails(tt.showDetails),
				WithShowErrors(tt.showErrors),
			)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)

			err := writer.Write(tt.result, tt.statusCode, rec, req)
			if err != nil {
				t.Fatalf("Write() error = %v", err)
			}

			// Check HTTP status code
			if rec.Code != tt.wantHTTPStatus {
				t.Errorf("HTTP status = %d, want %d", rec.Code, tt.wantHTTPStatus)
			}

			// Check Content-Type
			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/health+json" {
				t.Errorf("Content-Type = %q, want %q", contentType, "application/health+json")
			}

			// Parse response
			var resp ietfResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Check status
			if resp.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", resp.Status, tt.wantStatus)
			}

			// Check checks presence
			if tt.wantChecks && len(resp.Checks) == 0 {
				t.Error("expected checks in response, got none")
			}
			if !tt.wantChecks && len(resp.Checks) > 0 {
				t.Error("expected no checks in response, got some")
			}
		})
	}
}

func TestIETFResultWriter_CheckDetails(t *testing.T) {
	fixedTime := time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC)

	result := &CheckerResult{
		Status: StatusDown,
		Details: map[string]CheckResult{
			"db":    {Status: StatusUp, Timestamp: fixedTime},
			"cache": {Status: StatusDown, Timestamp: fixedTime, Error: errors.New("connection refused")},
		},
	}

	t.Run("details with errors shown", func(t *testing.T) {
		writer := NewIETFResultWriter(
			WithShowDetails(true),
			WithShowErrors(true),
		)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)

		err := writer.Write(result, http.StatusServiceUnavailable, rec, req)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		var resp ietfResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check db check
		if dbChecks, ok := resp.Checks["db"]; !ok || len(dbChecks) == 0 {
			t.Fatal("expected db check in response")
		} else {
			dbCheck := dbChecks[0]
			if dbCheck.Status != "pass" {
				t.Errorf("db status = %q, want %q", dbCheck.Status, "pass")
			}
			if dbCheck.Time != fixedTime.Format(time.RFC3339) {
				t.Errorf("db time = %q, want %q", dbCheck.Time, fixedTime.Format(time.RFC3339))
			}
			if dbCheck.Output != "" {
				t.Errorf("db output = %q, want empty (no error)", dbCheck.Output)
			}
		}

		// Check cache check
		if cacheChecks, ok := resp.Checks["cache"]; !ok || len(cacheChecks) == 0 {
			t.Fatal("expected cache check in response")
		} else {
			cacheCheck := cacheChecks[0]
			if cacheCheck.Status != "fail" {
				t.Errorf("cache status = %q, want %q", cacheCheck.Status, "fail")
			}
			if cacheCheck.Output != "connection refused" {
				t.Errorf("cache output = %q, want %q", cacheCheck.Output, "connection refused")
			}
		}
	})

	t.Run("details without errors", func(t *testing.T) {
		writer := NewIETFResultWriter(
			WithShowDetails(true),
			WithShowErrors(false), // Errors hidden
		)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)

		err := writer.Write(result, http.StatusServiceUnavailable, rec, req)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}

		var resp ietfResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Cache check should have no output (errors hidden)
		if cacheChecks, ok := resp.Checks["cache"]; !ok || len(cacheChecks) == 0 {
			t.Fatal("expected cache check in response")
		} else {
			cacheCheck := cacheChecks[0]
			if cacheCheck.Output != "" {
				t.Errorf("cache output = %q, want empty (errors hidden)", cacheCheck.Output)
			}
		}
	})
}

func TestMapStatusToIETF(t *testing.T) {
	tests := []struct {
		status AvailabilityStatus
		want   string
	}{
		{StatusUp, "pass"},
		{StatusDown, "fail"},
		{StatusUnknown, "warn"},
		{"invalid", "warn"}, // Unknown status defaults to warn
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := mapStatusToIETF(tt.status)
			if got != tt.want {
				t.Errorf("mapStatusToIETF(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}
