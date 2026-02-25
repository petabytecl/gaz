package gaz

import (
	"context"
	"testing"
	"time"

	"github.com/petabytecl/gaz/di"
)

type mockService struct {
	started bool
}

func (m *mockService) OnStart(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *mockService) OnStop(ctx context.Context) error {
	m.started = false
	return nil
}

// BenchmarkBuild benchmarks the Build() operation.
func BenchmarkBuild(b *testing.B) {
	for i := 0; i < b.N; i++ {
		app := New()
		For[*mockService](app.Container()).Provider(func(*di.Container) (*mockService, error) {
			return &mockService{}, nil
		})
		_ = app.Build()
	}
}

// BenchmarkStartup benchmarks service startup lifecycle.
func BenchmarkStartup(b *testing.B) {
	app := New()
	For[*mockService](app.Container()).Provider(func(*di.Container) (*mockService, error) {
		return &mockService{}, nil
	})
	if err := app.Build(); err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = app.Run(ctx)
		_ = app.Stop(ctx)
	}
}

// BenchmarkShutdown benchmarks service shutdown lifecycle.
func BenchmarkShutdown(b *testing.B) {
	app := New()
	For[*mockService](app.Container()).Provider(func(*di.Container) (*mockService, error) {
		return &mockService{}, nil
	})
	if err := app.Build(); err != nil {
		b.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Start app in background
	go func() {
		_ = app.Run(ctx)
	}()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = app.Stop(ctx)
	}
}
