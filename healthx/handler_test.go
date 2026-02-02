package healthx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHandler(t *testing.T) {
	t.Run("returns 200 when all checks pass", func(t *testing.T) {
		checker := NewChecker(
			WithCheck(Check{
				Name:  "db",
				Check: func(ctx context.Context) error { return nil },
			}),
		)

		handler := NewHandler(checker)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var resp ietfResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		if resp.Status != "pass" {
			t.Errorf("status = %q, want %q", resp.Status, "pass")
		}
	})

	t.Run("returns 503 when check fails", func(t *testing.T) {
		checker := NewChecker(
			WithCheck(Check{
				Name:  "db",
				Check: func(ctx context.Context) error { return errors.New("connection failed") },
			}),
		)

		handler := NewHandler(checker)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}

		var resp ietfResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		if resp.Status != "fail" {
			t.Errorf("status = %q, want %q", resp.Status, "fail")
		}
	})

	t.Run("returns 503 for unknown status", func(t *testing.T) {
		// Checker with no checks returns StatusUnknown
		checker := NewChecker()

		handler := NewHandler(checker)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})
}

func TestHandler_WithStatusCodeUp(t *testing.T) {
	checker := NewChecker(
		WithCheck(Check{
			Name:  "db",
			Check: func(ctx context.Context) error { return nil },
		}),
	)

	handler := NewHandler(checker, WithStatusCodeUp(http.StatusAccepted))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
}

func TestHandler_WithStatusCodeDown(t *testing.T) {
	checker := NewChecker(
		WithCheck(Check{
			Name:  "db",
			Check: func(ctx context.Context) error { return errors.New("fail") },
		}),
	)

	handler := NewHandler(checker, WithStatusCodeDown(http.StatusBadGateway))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

func TestHandler_LivenessPattern(t *testing.T) {
	// Liveness pattern: return 200 even on failure
	// Body still contains actual status for logging/debugging
	checker := NewChecker(
		WithCheck(Check{
			Name:  "db",
			Check: func(ctx context.Context) error { return errors.New("connection failed") },
		}),
	)

	handler := NewHandler(checker,
		WithStatusCodeUp(http.StatusOK),
		WithStatusCodeDown(http.StatusOK), // 200 on failure for liveness
	)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)

	handler.ServeHTTP(rec, req)

	// Should return 200 even though check failed
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (liveness should return 200)", rec.Code, http.StatusOK)
	}

	// Body should still indicate failure
	var resp ietfResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if resp.Status != "fail" {
		t.Errorf("body status = %q, want %q (body should still indicate failure)", resp.Status, "fail")
	}
}

func TestHandler_WithResultWriter(t *testing.T) {
	checker := NewChecker(
		WithCheck(Check{
			Name:  "db",
			Check: func(ctx context.Context) error { return nil },
		}),
	)

	// Use result writer with details enabled
	writer := NewIETFResultWriter(WithShowDetails(true))

	handler := NewHandler(checker, WithResultWriter(writer))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.ServeHTTP(rec, req)

	var resp ietfResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should have checks since details enabled
	if len(resp.Checks) == 0 {
		t.Error("expected checks in response when details enabled")
	}
	if _, ok := resp.Checks["db"]; !ok {
		t.Error("expected db check in response")
	}
}

func TestHandler_ContextPropagation(t *testing.T) {
	// Verify context from request is passed to checker
	type ctxKey string
	var gotContext context.Context

	checker := NewChecker(
		WithCheck(Check{
			Name: "ctx-check",
			Check: func(ctx context.Context) error {
				gotContext = ctx
				return nil
			},
		}),
	)

	handler := NewHandler(checker)

	// Create request with custom context value
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	ctx := context.WithValue(req.Context(), ctxKey("test"), "value")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Verify context was propagated
	if gotContext == nil {
		t.Fatal("context was not propagated to checker")
	}
	if val, ok := gotContext.Value(ctxKey("test")).(string); !ok || val != "value" {
		t.Error("context value not propagated correctly")
	}
}

func TestHandler_IntegrationWithRealChecker(t *testing.T) {
	// Integration test with real checker configuration
	checker := NewChecker(
		WithTimeout(2*time.Second),
		WithCheck(Check{
			Name: "fast-check",
			Check: func(ctx context.Context) error {
				return nil
			},
		}),
		WithCheck(Check{
			Name: "slow-check",
			Check: func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(100 * time.Millisecond):
					return nil
				}
			},
		}),
	)

	handler := NewHandler(checker,
		WithResultWriter(NewIETFResultWriter(WithShowDetails(true))),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Check Content-Type header
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/health+json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/health+json")
	}

	var resp ietfResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Status != "pass" {
		t.Errorf("status = %q, want %q", resp.Status, "pass")
	}
	if len(resp.Checks) != 2 {
		t.Errorf("expected 2 checks, got %d", len(resp.Checks))
	}
}

func TestHandler_ContentTypeHeader(t *testing.T) {
	checker := NewChecker(
		WithCheck(Check{
			Name:  "db",
			Check: func(ctx context.Context) error { return nil },
		}),
	)

	handler := NewHandler(checker)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	handler.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/health+json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/health+json")
	}
}
