# Testing with gaz

This guide covers testing patterns for gaz applications.

## Quick Reference

```go
// Basic test app setup
app, err := gaztest.New(t).Build()
app.RequireStart()
defer app.RequireStop()

// With modules (v3 pattern)
app, err := gaztest.New(t).
    WithModules(myModule).
    Build()

// Type-safe resolution that fails on error
svc := gaztest.RequireResolve[*MyService](t, app)

// Per-subsystem helpers
cfg := health.TestConfig()
worker := worker.NewMockWorker()
job := cron.NewMockJob("my-job")
bus := eventbus.TestBus()
mgr := config.TestManager(map[string]any{"key": "value"})
```

## Testing Patterns

### Unit Testing with Mocks

When testing a single component in isolation, use mock dependencies:

```go
func TestUserService_Create(t *testing.T) {
    // Create mocks
    mockDB := &mocks.Database{}
    mockDB.On("Insert", mock.Anything).Return(nil)
    
    // Build test app with mock
    baseApp := gaz.New()
    gaz.For[Database](baseApp.Container()).Instance(mockDB)
    baseApp.Build()
    
    app, err := gaztest.New(t).
        WithApp(baseApp).
        Build()
    require.NoError(t, err)
    
    app.RequireStart()
    defer app.RequireStop()
    
    // Resolve service under test
    svc := gaztest.RequireResolve[*UserService](t, app)
    
    // Test
    err = svc.Create(ctx, user)
    require.NoError(t, err)
    mockDB.AssertExpectations(t)
}
```

### Integration Testing with Modules

When testing module interactions, use WithModules:

```go
func TestHealthModule_Integration(t *testing.T) {
    cfg := health.TestConfig()
    module := health.NewModule(health.WithConfig(cfg))
    
    app, err := gaztest.New(t).
        WithModules(module).
        Build()
    require.NoError(t, err)
    
    app.RequireStart()
    defer app.RequireStop()
    
    // Verify health endpoints are available
    mgr := gaztest.RequireResolve[*health.Manager](t, app)
    require.NotNil(t, mgr)
}
```

### Testing Workers

```go
func TestWorkerLifecycle(t *testing.T) {
    w := worker.NewSimpleWorker("test-worker")
    
    // Register and start
    mgr := worker.TestManager(nil)
    mgr.Add(w)
    
    ctx := context.Background()
    require.NoError(t, mgr.Start(ctx))
    
    // Assert
    worker.RequireWorkerStarted(t, w)
    
    // Cleanup
    require.NoError(t, mgr.Stop(ctx))
    worker.RequireWorkerStopped(t, w)
}
```

### Testing Cron Jobs

```go
func TestCronJob_Execution(t *testing.T) {
    job := cron.NewSimpleJob("test-job", "@every 1s")
    
    // Manually invoke Run to test job logic
    err := job.Run(context.Background())
    require.NoError(t, err)
    
    cron.RequireJobRan(t, job)
    cron.RequireJobRunCount(t, job, 1)
}
```

### Testing EventBus Subscribers

```go
func TestEventHandler(t *testing.T) {
    bus := eventbus.TestBus()
    defer bus.Close()
    
    // Create subscriber expecting 1 event
    sub := eventbus.NewTestSubscriber[UserCreated](1)
    eventbus.Subscribe(bus, sub.Handler())
    
    // Publish
    eventbus.Publish(context.Background(), bus, UserCreated{UserID: "123"}, "")
    
    // Wait for async delivery
    eventbus.RequireEventsReceived(t, sub, time.Second)
    eventbus.RequireEventCount(t, sub, 1)
    
    // Assert event content
    events := sub.Events()
    require.Equal(t, "123", events[0].UserID)
}
```

### Testing Configuration

```go
func TestConfigLoading(t *testing.T) {
    mgr := config.TestManager(map[string]any{
        "app.host": "localhost",
        "app.port": 9090,
    })
    
    var cfg config.SampleConfig
    config.RequireConfigLoaded(t, mgr, &cfg)
    
    require.Equal(t, "localhost", cfg.Host)
    require.Equal(t, 9090, cfg.Port)
}
```

## Unit vs Integration Testing

| Pattern | When to Use | Tools |
|---------|-------------|-------|
| Unit (mocks) | Testing single component logic | MockWorker, MockJob, MockRegistrar |
| Integration | Testing component interactions | WithModules, real subsystems |
| End-to-end | Testing full app behavior | WithApp, full lifecycle |

### Guidelines

1. **Prefer unit tests** for business logic
2. **Use integration tests** for module wiring verification  
3. **Mock external dependencies** (databases, APIs)
4. **Use TestConfig/TestManager** for subsystem defaults
5. **Use RequireResolve** instead of manual Resolve + error check

## Subsystem Test Helpers

Each subsystem provides test helpers in a `testing.go` file:

- `health/testing.go` - TestConfig, NewTestConfig, MockRegistrar, TestManager, RequireHealthy, RequireLivenessCheckRegistered
- `worker/testing.go` - MockWorker, NewMockWorker, SimpleWorker, TestManager, RequireWorkerStarted, RequireWorkerStopped
- `cron/testing.go` - MockJob, SimpleJob, MockResolver, TestScheduler, RequireJobRan, RequireJobRunCount
- `config/testing.go` - MapBackend, TestManager, SampleConfig, RequireConfigLoaded, RequireConfigValue
- `eventbus/testing.go` - TestBus, TestSubscriber, TestEvent, RequireEventsReceived, RequireEventCount

## Best Practices

1. **Always use t.Cleanup or defer** for app shutdown
2. **Use RequireResolve** for cleaner test code
3. **TestSubscriber.WaitFor** handles async eventbus delivery
4. **Port 0** in TestConfig selects random available port
5. **SimpleWorker/SimpleJob** are easier than mocks for simple cases
