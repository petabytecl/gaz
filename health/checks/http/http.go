// Package http provides a health check for HTTP upstream services.
package http

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Config configures the HTTP upstream health check.
type Config struct {
	// URL is the health check endpoint URL. Required.
	URL string
	// Timeout for the HTTP request. Optional, defaults to 5s.
	// The context deadline takes precedence if shorter.
	Timeout time.Duration
	// ExpectedStatusCode is the expected response status. Optional, defaults to 200.
	ExpectedStatusCode int
	// Client is the HTTP client to use. Optional, a default client is created.
	// Providing a custom client allows reusing connection pools and custom TLS config.
	Client *http.Client
}

// New creates a new HTTP upstream health check.
// Performs GET request and validates response status code.
//
// Returns nil if response matches expected status, error otherwise.
func New(cfg Config) func(context.Context) error {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.ExpectedStatusCode == 0 {
		cfg.ExpectedStatusCode = http.StatusOK
	}

	client := cfg.Client
	if client == nil {
		client = &http.Client{
			Timeout: cfg.Timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		}
	}

	return func(ctx context.Context) error {
		if cfg.URL == "" {
			return fmt.Errorf("http: URL is empty")
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
		if err != nil {
			return fmt.Errorf("http: failed to create request: %w", err)
		}
		req.Header.Set("Connection", "close")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("http: request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != cfg.ExpectedStatusCode {
			return fmt.Errorf("http: unexpected status %d (expected %d)",
				resp.StatusCode, cfg.ExpectedStatusCode)
		}
		return nil
	}
}
