package health

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/petabytecl/gaz/health/internal/healthx"
)

func TestIETFResultWriter(t *testing.T) {
	// Setup a sample result
	timestamp := time.Date(2023, 10, 26, 12, 0, 0, 0, time.UTC)
	result := &healthx.CheckerResult{
		Status: healthx.StatusDown,
		Details: map[string]healthx.CheckResult{
			"db": {
				Status:    healthx.StatusUp,
				Timestamp: timestamp,
			},
			"redis": {
				Status:    healthx.StatusDown,
				Timestamp: timestamp,
				Error:     errors.New("connection refused"),
			},
		},
	}

	// Create a recorder
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)

	// Call the writer with options to show details and errors
	writer := NewIETFResultWriter(
		healthx.WithShowDetails(true),
		healthx.WithShowErrors(true),
	)
	err := writer.Write(result, http.StatusServiceUnavailable, w, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify status code
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	// Verify Content-Type
	if ct := w.Header().Get("Content-Type"); ct != "application/health+json" {
		t.Errorf("expected Content-Type application/health+json, got %s", ct)
	}

	// Verify JSON body
	var body map[string]any
	if unmarshalErr := json.Unmarshal(w.Body.Bytes(), &body); unmarshalErr != nil {
		t.Fatalf("failed to unmarshal body: %v", unmarshalErr)
	}

	// Check root status
	if body["status"] != "fail" {
		t.Errorf("expected status fail, got %v", body["status"])
	}

	// Check checks
	checks, ok := body["checks"].(map[string]any)
	if !ok {
		t.Fatalf("checks field missing or invalid")
	}

	// Check DB
	dbChecks, ok := checks["db"].([]any)
	if !ok || len(dbChecks) != 1 {
		t.Fatalf("db checks missing or invalid")
	}
	dbCheck, ok := dbChecks[0].(map[string]any)
	if !ok {
		t.Fatalf("db check result is not a map")
	}
	if dbCheck["status"] != "pass" {
		t.Errorf("expected db status pass, got %v", dbCheck["status"])
	}
	if dbCheck["time"] != timestamp.Format(time.RFC3339) {
		t.Errorf("expected db time %s, got %v", timestamp.Format(time.RFC3339), dbCheck["time"])
	}

	// Check Redis
	redisChecks, ok := checks["redis"].([]any)
	if !ok || len(redisChecks) != 1 {
		t.Fatalf("redis checks missing or invalid")
	}
	redisCheck, ok := redisChecks[0].(map[string]any)
	if !ok {
		t.Fatalf("redis check result is not a map")
	}
	if redisCheck["status"] != "fail" {
		t.Errorf("expected redis status fail, got %v", redisCheck["status"])
	}
	if redisCheck["output"] != "connection refused" {
		t.Errorf("expected redis output 'connection refused', got %v", redisCheck["output"])
	}
}
