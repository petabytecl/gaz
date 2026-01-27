package health

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexliesenfeld/health"
)

func TestIETFResultWriter(t *testing.T) {
	// Setup a sample result
	timestamp := time.Date(2023, 10, 26, 12, 0, 0, 0, time.UTC)
	result := &health.CheckerResult{
		Status: health.StatusDown,
		Details: map[string]health.CheckResult{
			"db": {
				Status:    health.StatusUp,
				Timestamp: timestamp,
			},
			"redis": {
				Status:    health.StatusDown,
				Timestamp: timestamp,
				Error:     errors.New("connection refused"),
			},
		},
	}

	// Create a recorder
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)

	// Call the writer
	writer := NewIETFResultWriter()
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
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}

	// Check root status
	if body["status"] != "fail" {
		t.Errorf("expected status fail, got %v", body["status"])
	}

	// Check checks
	checks, ok := body["checks"].(map[string]interface{})
	if !ok {
		t.Fatalf("checks field missing or invalid")
	}

	// Check DB
	dbChecks, ok := checks["db"].([]interface{})
	if !ok || len(dbChecks) != 1 {
		t.Fatalf("db checks missing or invalid")
	}
	dbCheck := dbChecks[0].(map[string]interface{})
	if dbCheck["status"] != "pass" {
		t.Errorf("expected db status pass, got %v", dbCheck["status"])
	}
	if dbCheck["time"] != timestamp.Format(time.RFC3339) {
		t.Errorf("expected db time %s, got %v", timestamp.Format(time.RFC3339), dbCheck["time"])
	}

	// Check Redis
	redisChecks, ok := checks["redis"].([]interface{})
	if !ok || len(redisChecks) != 1 {
		t.Fatalf("redis checks missing or invalid")
	}
	redisCheck := redisChecks[0].(map[string]interface{})
	if redisCheck["status"] != "fail" {
		t.Errorf("expected redis status fail, got %v", redisCheck["status"])
	}
	if redisCheck["output"] != "connection refused" {
		t.Errorf("expected redis output 'connection refused', got %v", redisCheck["output"])
	}
}
