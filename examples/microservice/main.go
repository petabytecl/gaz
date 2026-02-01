// Package main demonstrates a complete microservice with gaz.
//
// This example shows:
//   - Health check endpoints (/live, /ready)
//   - Background worker for event processing
//   - Event bus for internal messaging
//   - Full lifecycle management
//
// Run with: go run .
// Check health: curl http://localhost:9090/ready
// Stop with: Ctrl+C (SIGINT)
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/eventbus"
	"github.com/petabytecl/gaz/health"
	"github.com/petabytecl/gaz/worker"
)

// --- Events ---

// OrderCreatedEvent is published when a new order is placed.
type OrderCreatedEvent struct {
	OrderID    string
	CustomerID string
	Amount     float64
	CreatedAt  time.Time
}

// EventName returns the event identifier for logging.
func (e OrderCreatedEvent) EventName() string { return "OrderCreated" }

// OrderProcessedEvent is published when an order has been processed.
type OrderProcessedEvent struct {
	OrderID     string
	ProcessedAt time.Time
}

// EventName returns the event identifier for logging.
func (e OrderProcessedEvent) EventName() string { return "OrderProcessed" }

// Compile-time interface checks.
var (
	_ eventbus.Event = OrderCreatedEvent{}
	_ eventbus.Event = OrderProcessedEvent{}
)

// --- Order Processor Worker ---

// OrderProcessor handles order events in the background.
// It subscribes to OrderCreatedEvent and processes each order.
type OrderProcessor struct {
	bus  *eventbus.EventBus
	sub  *eventbus.Subscription
	done chan struct{}
	wg   sync.WaitGroup
}

// NewOrderProcessor creates a new order processor.
func NewOrderProcessor(bus *eventbus.EventBus) *OrderProcessor {
	return &OrderProcessor{
		bus: bus,
	}
}

// Name returns the worker's unique identifier.
func (p *OrderProcessor) Name() string { return "order-processor" }

// OnStart subscribes to order events and starts processing.
func (p *OrderProcessor) OnStart(ctx context.Context) error {
	fmt.Printf("[%s] starting\n", p.Name())

	p.done = make(chan struct{})

	// Subscribe to OrderCreatedEvent
	p.sub = eventbus.Subscribe[OrderCreatedEvent](p.bus, func(ctx context.Context, evt OrderCreatedEvent) {
		fmt.Printf("[%s] processing order %s: $%.2f for customer %s\n",
			p.Name(), evt.OrderID, evt.Amount, evt.CustomerID)

		// Simulate processing time
		time.Sleep(500 * time.Millisecond)

		// Publish processed event
		eventbus.Publish(ctx, p.bus, OrderProcessedEvent{
			OrderID:     evt.OrderID,
			ProcessedAt: time.Now(),
		}, "")

		fmt.Printf("[%s] order %s processed\n", p.Name(), evt.OrderID)
	})

	return nil
}

// OnStop gracefully shuts down the processor.
func (p *OrderProcessor) OnStop(ctx context.Context) error {
	fmt.Printf("[%s] stopping...\n", p.Name())

	if p.sub != nil {
		p.sub.Unsubscribe()
	}

	fmt.Printf("[%s] stopped\n", p.Name())
	return nil
}

// Compile-time interface check.
var _ worker.Worker = (*OrderProcessor)(nil)

// --- Order Simulator Worker ---

// OrderSimulator creates fake orders for demonstration.
type OrderSimulator struct {
	bus  *eventbus.EventBus
	done chan struct{}
	wg   sync.WaitGroup
}

// NewOrderSimulator creates a new order simulator.
func NewOrderSimulator(bus *eventbus.EventBus) *OrderSimulator {
	return &OrderSimulator{bus: bus}
}

func (s *OrderSimulator) Name() string { return "order-simulator" }

func (s *OrderSimulator) OnStart(ctx context.Context) error {
	fmt.Printf("[%s] starting\n", s.Name())

	s.done = make(chan struct{})
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		orderNum := 1

		for {
			select {
			case <-ctx.Done():
				fmt.Printf("[%s] context cancelled\n", s.Name())
				return
			case <-s.done:
				fmt.Printf("[%s] received stop signal\n", s.Name())
				return
			case <-ticker.C:
				orderID := fmt.Sprintf("ORD-%03d", orderNum)
				orderNum++

				evt := OrderCreatedEvent{
					OrderID:    orderID,
					CustomerID: fmt.Sprintf("CUST-%03d", orderNum%10),
					Amount:     float64(orderNum * 25),
					CreatedAt:  time.Now(),
				}

				fmt.Printf("[%s] creating order %s\n", s.Name(), orderID)
				eventbus.Publish(ctx, s.bus, evt, "")
			}
		}
	}()

	return nil
}

func (s *OrderSimulator) OnStop(ctx context.Context) error {
	fmt.Printf("[%s] stopping...\n", s.Name())
	close(s.done)
	s.wg.Wait()
	fmt.Printf("[%s] stopped\n", s.Name())
	return nil
}

var _ worker.Worker = (*OrderSimulator)(nil)

// --- Notification Subscriber ---

// NotificationSubscriber logs when orders are processed.
// This demonstrates multiple subscribers to the same event type.
type NotificationSubscriber struct {
	bus *eventbus.EventBus
	sub *eventbus.Subscription
}

// NewNotificationSubscriber creates a new notification subscriber.
func NewNotificationSubscriber(bus *eventbus.EventBus) *NotificationSubscriber {
	return &NotificationSubscriber{bus: bus}
}

func (n *NotificationSubscriber) Name() string { return "notification-subscriber" }

func (n *NotificationSubscriber) OnStart(ctx context.Context) error {
	fmt.Printf("[%s] starting\n", n.Name())

	// Subscribe to OrderProcessedEvent
	n.sub = eventbus.Subscribe[OrderProcessedEvent](n.bus, func(ctx context.Context, evt OrderProcessedEvent) {
		fmt.Printf("[%s] sending notification: order %s processed at %s\n",
			n.Name(), evt.OrderID, evt.ProcessedAt.Format("15:04:05"))
	})

	return nil
}

func (n *NotificationSubscriber) OnStop(ctx context.Context) error {
	fmt.Printf("[%s] stopping...\n", n.Name())
	if n.sub != nil {
		n.sub.Unsubscribe()
	}
	fmt.Printf("[%s] stopped\n", n.Name())
	return nil
}

var _ worker.Worker = (*NotificationSubscriber)(nil)

func main() {
	app := gaz.New()

	// Register modules
	// Health module: provides /live, /ready, /startup endpoints on port 9090
	app.UseDI(health.NewModule(health.WithPort(9090)))

	// Worker module: validates worker prerequisites
	app.UseDI(worker.NewModule())

	// EventBus module: provides pub/sub infrastructure
	app.UseDI(eventbus.NewModule())

	// Register OrderProcessor (eager = auto-start)
	if err := gaz.For[*OrderProcessor](app.Container()).
		Eager().
		Provider(func(c *gaz.Container) (*OrderProcessor, error) {
			bus, err := gaz.Resolve[*eventbus.EventBus](c)
			if err != nil {
				return nil, err
			}
			return NewOrderProcessor(bus), nil
		}); err != nil {
		log.Fatal(err)
	}

	// Register OrderSimulator (eager = auto-start)
	if err := gaz.For[*OrderSimulator](app.Container()).
		Eager().
		Provider(func(c *gaz.Container) (*OrderSimulator, error) {
			bus, err := gaz.Resolve[*eventbus.EventBus](c)
			if err != nil {
				return nil, err
			}
			return NewOrderSimulator(bus), nil
		}); err != nil {
		log.Fatal(err)
	}

	// Register NotificationSubscriber (eager = auto-start)
	if err := gaz.For[*NotificationSubscriber](app.Container()).
		Eager().
		Provider(func(c *gaz.Container) (*NotificationSubscriber, error) {
			bus, err := gaz.Resolve[*eventbus.EventBus](c)
			if err != nil {
				return nil, err
			}
			return NewNotificationSubscriber(bus), nil
		}); err != nil {
		log.Fatal(err)
	}

	// Build the application
	if err := app.Build(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("=================================================")
	fmt.Println("Microservice starting...")
	fmt.Println("Health check: http://localhost:9090/ready")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println("=================================================")

	// Run blocks until shutdown signal
	if err := app.Run(context.Background()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Shutdown complete")
}
