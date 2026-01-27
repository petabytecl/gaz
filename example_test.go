package gaz_test

import (
	"fmt"

	"github.com/petabytecl/gaz"
)

// MyService is a simple service for demonstration.
type MyService struct {
	Name string
}

// Logger is a service that other services depend on.
type Logger struct {
	Level string
}

// UserService demonstrates dependency injection.
type UserService struct {
	logger *Logger
}

// Counter tracks resolution calls for singleton vs transient demo.
type Counter struct {
	value int
}

func (c *Counter) Next() int {
	c.value++
	return c.value
}

// ExampleNew demonstrates basic app creation and provider registration.
func ExampleNew() {
	app := gaz.New()
	app.ProvideSingleton(func(c *gaz.Container) (*MyService, error) {
		return &MyService{Name: "example"}, nil
	})
	if err := app.Build(); err != nil {
		fmt.Println("error:", err)
		return
	}
	svc, err := gaz.Resolve[*MyService](app.Container())
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(svc.Name)
	// Output: example
}

// ExampleFor_singleton demonstrates registering a singleton service.
// Singletons return the same instance on every resolution.
func ExampleFor_singleton() {
	c := gaz.NewContainer()

	counter := &Counter{}
	err := gaz.For[*Counter](c).Instance(counter)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Resolve twice - same instance returned
	c1, _ := gaz.Resolve[*Counter](c)
	c2, _ := gaz.Resolve[*Counter](c)

	fmt.Println("same instance:", c1 == c2)
	// Output: same instance: true
}

// ExampleFor_transient demonstrates registering a transient service.
// Transient services return a new instance on every resolution.
func ExampleFor_transient() {
	c := gaz.NewContainer()

	callCount := 0
	err := gaz.For[*Counter](c).Transient().Provider(func(_ *gaz.Container) (*Counter, error) {
		callCount++
		return &Counter{value: callCount * 100}, nil
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Resolve twice - different instances created
	c1, _ := gaz.Resolve[*Counter](c)
	c2, _ := gaz.Resolve[*Counter](c)

	fmt.Println("different instances:", c1 != c2)
	fmt.Println("c1 value:", c1.value)
	fmt.Println("c2 value:", c2.value)
	// Output:
	// different instances: true
	// c1 value: 100
	// c2 value: 200
}

// ExampleResolve demonstrates resolving a registered service.
func ExampleResolve() {
	c := gaz.NewContainer()

	// Register a configuration instance
	err := gaz.For[*Logger](c).Instance(&Logger{Level: "debug"})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Resolve the service
	logger, err := gaz.Resolve[*Logger](c)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("level:", logger.Level)
	// Output: level: debug
}

// ExampleApp_ProvideSingleton demonstrates dependency injection with App.
// The provider function receives the Container for resolving dependencies.
func ExampleApp_ProvideSingleton() {
	app := gaz.New()

	// Register Logger first
	app.ProvideSingleton(func(c *gaz.Container) (*Logger, error) {
		return &Logger{Level: "info"}, nil
	})

	// UserService depends on Logger
	app.ProvideSingleton(func(c *gaz.Container) (*UserService, error) {
		logger, err := gaz.Resolve[*Logger](c)
		if err != nil {
			return nil, err
		}
		return &UserService{logger: logger}, nil
	})

	if err := app.Build(); err != nil {
		fmt.Println("error:", err)
		return
	}

	svc, _ := gaz.Resolve[*UserService](app.Container())
	fmt.Println("logger level:", svc.logger.Level)
	// Output: logger level: info
}

// ExampleContainer_Build demonstrates the container build process.
// Eager services are instantiated at Build() time.
func ExampleContainer_Build() {
	c := gaz.NewContainer()

	instantiated := false
	err := gaz.For[*MyService](c).Eager().Provider(func(_ *gaz.Container) (*MyService, error) {
		instantiated = true
		return &MyService{Name: "eager"}, nil
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("before build:", instantiated)

	if err := c.Build(); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("after build:", instantiated)
	// Output:
	// before build: false
	// after build: true
}
