package gateway

import (
	"net/http"
	"sync/atomic"
)

// DynamicHandler is an http.Handler that delegates to an atomic underlying handler.
// This allows the handler to be swapped at runtime (e.g. during OnStart) without
// race conditions, and provides a safe handler to register before the Gateway starts.
type DynamicHandler struct {
	handler atomic.Value
}

// NewDynamicHandler creates a new DynamicHandler with an optional initial handler.
// If initial is nil, it defaults to http.NotFoundHandler().
func NewDynamicHandler(initial http.Handler) *DynamicHandler {
	dh := &DynamicHandler{}
	if initial == nil {
		initial = http.NotFoundHandler()
	}
	dh.handler.Store(initial)
	return dh
}

// SetHandler updates the underlying handler atomically.
func (h *DynamicHandler) SetHandler(handler http.Handler) {
	if handler == nil {
		handler = http.NotFoundHandler()
	}
	h.handler.Store(handler)
}

// ServeHTTP delegates to the current underlying handler.
func (h *DynamicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.Load().(http.Handler).ServeHTTP(w, r)
}
