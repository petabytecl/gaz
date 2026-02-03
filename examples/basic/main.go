// Package main demonstrates the minimal usage of the gaz DI framework.
package main

import (
	"fmt"
	"log"

	"github.com/petabytecl/gaz"
)

// Greeter is a simple service that holds a greeting name.
type Greeter struct {
	Name string
}

func run() error {
	// Create a new application
	app := gaz.New()

	// Register a singleton provider for Greeter using For[T]()
	err := gaz.For[*Greeter](app.Container()).Provider(func(c *gaz.Container) (*Greeter, error) {
		return &Greeter{Name: "World"}, nil
	})
	if err != nil {
		return err
	}

	// Build the application (validates and prepares services)
	if err := app.Build(); err != nil {
		return err
	}

	// Resolve the Greeter service from the container
	greeter, err := gaz.Resolve[*Greeter](app.Container())
	if err != nil {
		return err
	}

	fmt.Printf("Hello, %s!\n", greeter.Name)
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
