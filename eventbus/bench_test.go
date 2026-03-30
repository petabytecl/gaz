package eventbus_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/petabytecl/gaz/eventbus"
)

// sink prevents compiler optimisation of benchmark results.
//
//nolint:gochecknoglobals // required for benchmark correctness
var sink any

// benchEvent implements eventbus.Event for benchmarks.
type benchEvent struct {
	Value int
}

func (e benchEvent) EventName() string { return "benchEvent" }

func benchLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func BenchmarkPublish_SingleSubscriber(b *testing.B) {
	b.ReportAllocs()

	bus := eventbus.New(benchLogger())
	defer bus.Close()

	eventbus.Subscribe(bus, func(_ context.Context, _ benchEvent) {
		// no-op handler
	})

	ctx := context.Background()
	evt := benchEvent{Value: 1}

	for b.Loop() {
		eventbus.Publish(ctx, bus, evt, "")
	}
}

func BenchmarkPublish_TenSubscribers(b *testing.B) {
	b.ReportAllocs()

	bus := eventbus.New(benchLogger())
	defer bus.Close()

	for range 10 {
		eventbus.Subscribe(bus, func(_ context.Context, _ benchEvent) {
			// no-op handler
		})
	}

	ctx := context.Background()
	evt := benchEvent{Value: 1}

	for b.Loop() {
		eventbus.Publish(ctx, bus, evt, "")
	}
}

func BenchmarkPublish_Parallel(b *testing.B) {
	b.ReportAllocs()

	bus := eventbus.New(benchLogger())
	defer bus.Close()

	eventbus.Subscribe(bus, func(_ context.Context, _ benchEvent) {
		// no-op handler
	})

	ctx := context.Background()
	evt := benchEvent{Value: 1}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			eventbus.Publish(ctx, bus, evt, "")
		}
	})
}
