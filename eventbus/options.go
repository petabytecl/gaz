package eventbus

// SubscribeOption configures a subscription.
//
// Options are passed to Subscribe to customize the subscription behavior.
// Use [WithTopic] to filter events by topic and [WithBufferSize] to control
// the async delivery buffer.
type SubscribeOption func(*subscribeOptions)

// subscribeOptions holds subscription configuration.
//
// These are internal options applied via functional option pattern.
type subscribeOptions struct {
	topic      string // Optional topic filter (empty = all topics)
	bufferSize int    // Buffer size for async delivery (default: 100)
}

// defaultSubscribeOptions returns the default subscription configuration.
//
// Defaults per RESEARCH.md:
//   - topic: "" (subscribe to all topics of this type)
//   - bufferSize: 100 (reasonable default for most use cases)
func defaultSubscribeOptions() subscribeOptions {
	return subscribeOptions{
		topic:      "", // Subscribe to all topics of this type
		bufferSize: 100,
	}
}

// WithTopic filters events to only those published with matching topic.
//
// When a topic is specified, the subscription only receives events that
// were published with the same topic string. This enables filtering a
// single event type by context (e.g., different user segments, regions).
//
// Empty topic or omitting this option subscribes to all events of the type.
//
// # Example
//
//	// Subscribe to all UserCreated events
//	eventbus.Subscribe[UserCreated](bus, handler)
//
//	// Subscribe only to admin UserCreated events
//	eventbus.Subscribe[UserCreated](bus, handler, eventbus.WithTopic("admin"))
func WithTopic(topic string) SubscribeOption {
	return func(o *subscribeOptions) {
		o.topic = topic
	}
}

// WithBufferSize sets the async buffer size for this subscription.
//
// Each subscription has its own buffered channel for async delivery.
// When the buffer is full, Publish blocks until space is available
// (backpressure). This prevents memory exhaustion from slow handlers.
//
// The default buffer size is 100. Increase for high-throughput handlers
// that process events quickly. Decrease for memory-constrained environments
// or handlers that should process events more synchronously.
//
// # Example
//
//	// High-throughput handler with large buffer
//	eventbus.Subscribe[OrderPlaced](bus, handler, eventbus.WithBufferSize(1000))
//
//	// Low-latency handler with small buffer
//	eventbus.Subscribe[Alert](bus, handler, eventbus.WithBufferSize(10))
func WithBufferSize(size int) SubscribeOption {
	return func(o *subscribeOptions) {
		o.bufferSize = size
	}
}

// applyOptions applies the given options to the default configuration.
//
// This is an internal helper used by Subscribe to merge options.
func applyOptions(opts []SubscribeOption) subscribeOptions {
	options := defaultSubscribeOptions()
	for _, opt := range opts {
		opt(&options)
	}
	return options
}
