package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlers(t *testing.T) {
	// 1. Setup Manager with failing checks
	m := NewManager()
	m.AddLivenessCheck("live_fail", func(ctx context.Context) error { return errors.New("fail") })
	m.AddReadinessCheck("ready_fail", func(ctx context.Context) error { return errors.New("fail") })
	m.AddStartupCheck("startup_fail", func(ctx context.Context) error { return errors.New("fail") })

	// 2. Test Liveness (Expect 200 on failure)
	t.Run("Liveness", func(t *testing.T) {
		h := m.NewLivenessHandler()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/live", nil)

		h.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
		if ct := w.Header().Get("Content-Type"); ct != "application/health+json" {
			t.Errorf("expected Content-Type application/health+json, got %s", ct)
		}
		if !strings.Contains(w.Body.String(), `"status":"fail"`) {
			t.Errorf("expected body to contain status:fail, got %s", w.Body.String())
		}
	})

	// 3. Test Readiness (Expect 503 on failure)
	t.Run("Readiness", func(t *testing.T) {
		h := m.NewReadinessHandler()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/ready", nil)

		h.ServeHTTP(w, r)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503 Service Unavailable, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), `"status":"fail"`) {
			t.Errorf("expected body to contain status:fail, got %s", w.Body.String())
		}
	})

	// 4. Test Startup (Expect 503 on failure)
	t.Run("Startup", func(t *testing.T) {
		h := m.NewStartupHandler()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/startup", nil)

		h.ServeHTTP(w, r)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected 503 Service Unavailable, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), `"status":"fail"`) {
			t.Errorf("expected body to contain status:fail, got %s", w.Body.String())
		}
	})
}

func TestHandlersSuccess(t *testing.T) {
	// 1. Setup Manager with passing checks
	m := NewManager()
	m.AddLivenessCheck("live_pass", func(ctx context.Context) error { return nil })

	// 2. Test Liveness (Expect 200)
	t.Run("LivenessSuccess", func(t *testing.T) {
		h := m.NewLivenessHandler()
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/live", nil)

		h.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), `"status":"pass"`) {
			t.Errorf("expected body to contain status:pass, got %s", w.Body.String())
		}
	})
}
