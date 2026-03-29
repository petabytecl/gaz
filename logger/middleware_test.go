package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsValidRequestID(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{"valid alphanumeric", "abc123", true},
		{"valid with dashes", "request-id-123", true},
		{"valid with underscores", "request_id_123", true},
		{"valid with dots", "request.id.123", true},
		{"valid max length 64", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
		{"too long 65 chars", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
		{"contains newline", "request\nid", false},
		{"contains semicolon", "request;id", false},
		{"contains angle bracket", "request<id>", false},
		{"contains space", "request id", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidRequestID(tt.id)
			if got != tt.valid {
				t.Errorf("isValidRequestID(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}

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

	t.Run("RejectsOversizedID", func(t *testing.T) {
		longID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 65 chars
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-ID", longID)
		w := httptest.NewRecorder()

		handler := RequestIDMiddleware(http.HandlerFunc(
			func(_ http.ResponseWriter, r *http.Request) {
				id := GetRequestID(r.Context())
				if id == longID {
					t.Error("oversized ID should have been replaced")
				}
				if len(id) != 32 {
					t.Errorf("expected generated 32-char ID, got %d chars", len(id))
				}
			}))

		handler.ServeHTTP(w, req)
	})

	t.Run("RejectsMalformedID", func(t *testing.T) {
		malformedIDs := []string{
			"request\nid",
			"request;id",
			"<script>alert(1)</script>",
			"id with spaces",
		}

		for _, malID := range malformedIDs {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-Request-ID", malID)
			w := httptest.NewRecorder()

			handler := RequestIDMiddleware(http.HandlerFunc(
				func(_ http.ResponseWriter, r *http.Request) {
					id := GetRequestID(r.Context())
					if id == malID {
						t.Errorf("malformed ID %q should have been replaced", malID)
					}
				}))

			handler.ServeHTTP(w, req)
		}
	})
}
