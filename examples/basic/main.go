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

func main() {
	// Create a new application
	app := gaz.New()

	// Register a singleton provider for Greeter
	app.ProvideSingleton(func(c *gaz.Container) (*Greeter, error) {
		return &Greeter{Name: "World"}, nil
	})

	// Build the application (validates and prepares services)
	if err := app.Build(); err != nil {
		log.Fatal(err)
	}

	// Resolve the Greeter service from the container
	greeter, err := gaz.Resolve[*Greeter](app.Container())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Hello, %s!\n", greeter.Name)
}
