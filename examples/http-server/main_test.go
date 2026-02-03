package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cfg := DefaultAppConfig()
	cfg.Server.Port = 0 // Let OS choose port
	cfg.Health.Port = 0 // Let OS choose port

	if err := run(ctx, cfg); err != nil {
		t.Fatalf("run() failed: %v", err)
	}
}

func TestHandlers(t *testing.T) {
	handler := NewHandler()

	// Test root endpoint
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, w.Body.String(), "http-server-example")

	// Test hello endpoint with name
	req = httptest.NewRequest("GET", "/hello?name=Test", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, w.Body.String(), "Hello, Test!")

	// Test hello endpoint default
	req = httptest.NewRequest("GET", "/hello", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp = w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, w.Body.String(), "Hello, World!")
}
