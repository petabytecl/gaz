package logger

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// generateID generates a random 16-byte hex string (32 characters).
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback or panic? For a logger middleware, fallback to empty or handled error is safer,
		// but rand.Read failing is catastrophic.
		// Let's just return a simpler fallback if this ever happens (unlikely).
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}

// RequestIDMiddleware checks for an incoming X-Request-ID header.
// If missing, it generates a new ID.
// It sets the X-Request-ID header on the response and adds the ID to the request context.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = generateID()
		}

		// Set the header on the response so the client knows the ID
		w.Header().Set("X-Request-ID", reqID)

		// Add request ID to context
		ctx := WithRequestID(r.Context(), reqID)

		// Serve next handler with new context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
