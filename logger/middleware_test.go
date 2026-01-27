package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddleware(t *testing.T) {
	t.Run("GeneratesIDIfMissing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler := RequestIDMiddleware(http.HandlerFunc(
			func(_ http.ResponseWriter, r *http.Request) {
				id := GetRequestID(r.Context())
				if id == "" {
					t.Error("context request ID is empty")
				}
				if len(id) != 32 { // 16 bytes hex = 32 chars
					t.Errorf("expected 32-char ID, got %d chars: %s", len(id), id)
				}
			}))

		handler.ServeHTTP(w, req)

		respID := w.Header().Get("X-Request-ID")
		if respID == "" {
			t.Error("response header X-Request-ID is empty")
		}
	})

	t.Run("PreservesIncomingID", func(t *testing.T) {
		existingID := "existing-request-id"
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-ID", existingID)
		w := httptest.NewRecorder()

		handler := RequestIDMiddleware(http.HandlerFunc(
			func(_ http.ResponseWriter, r *http.Request) {
				id := GetRequestID(r.Context())
				if id != existingID {
					t.Errorf("context ID = %s, want %s", id, existingID)
				}
			}))

		handler.ServeHTTP(w, req)

		respID := w.Header().Get("X-Request-ID")
		if respID != existingID {
			t.Errorf("response header ID = %s, want %s", respID, existingID)
		}
	})
}
