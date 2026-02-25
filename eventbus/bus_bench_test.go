package eventbus

import (
	"context"
	"log/slog"
	"testing"
)

type TestEvent struct {
	Value string
}

func (e TestEvent) EventName() string {
	return "test-event"
}

// BenchmarkPublish benchmarks publishing events to a single subscriber.
func BenchmarkPublish(b *testing.B) {
	bus := New(slog.Default())
	Subscribe[TestEvent](bus, func(ctx context.Context, event TestEvent) {
		_ = event.Value
	})

	event := TestEvent{Value: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Publish(context.Background(), bus, event, "")
	}
}

// BenchmarkSubscribe benchmarks subscription creation.
func BenchmarkSubscribe(b *testing.B) {
	bus := New(slog.Default())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Subscribe[TestEvent](bus, func(ctx context.Context, event TestEvent) {
			_ = event.Value
		})
	}
}

// BenchmarkPublishMultipleSubscribers benchmarks publishing to multiple subscribers.
func BenchmarkPublishMultipleSubscribers(b *testing.B) {
	bus := New(slog.Default())
	// Create 10 subscribers
	for i := 0; i < 10; i++ {
		Subscribe[TestEvent](bus, func(ctx context.Context, event TestEvent) {
			_ = event.Value
		})
	}

	event := TestEvent{Value: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Publish(context.Background(), bus, event, "")
	}
}
