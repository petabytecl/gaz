// Package main demonstrates organizing providers into modules.
//
// This example shows:
//   - Creating custom modules to group related providers
//   - Using app.Module() for named registration groups
//   - Module dependencies (services can depend on other module's services)
//   - Clean separation of concerns
//
// Run with: go run .
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/petabytecl/gaz"
)

// --- Database Module ---

// Database represents a database connection.
type Database struct {
	DSN       string
	Connected bool
}

// NewDatabase creates a new database connection.
func NewDatabase() *Database {
	return &Database{
		DSN:       "postgres://localhost:5432/myapp",
		Connected: true,
	}
}

// Query executes a database query.
func (db *Database) Query(sql string) ([]string, error) {
	if !db.Connected {
		return nil, fmt.Errorf("database not connected")
	}
	// Simulated query
	return []string{"row1", "row2", "row3"}, nil
}

// UserRepository handles user data access.
type UserRepository struct {
	db *Database
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *Database) *UserRepository {
	return &UserRepository{db: db}
}

// FindAll returns all users.
func (r *UserRepository) FindAll() ([]string, error) {
	return r.db.Query("SELECT * FROM users")
}

// --- Cache Module ---

// Cache provides in-memory caching.
type Cache struct {
	store map[string]string
}

// NewCache creates a new cache.
func NewCache() *Cache {
	return &Cache{
		store: make(map[string]string),
	}
}

// Get retrieves a value from cache.
func (c *Cache) Get(key string) (string, bool) {
	val, ok := c.store[key]
	return val, ok
}

// Set stores a value in cache.
func (c *Cache) Set(key, value string) {
	c.store[key] = value
}

// CachedUserRepository wraps UserRepository with caching.
type CachedUserRepository struct {
	repo  *UserRepository
	cache *Cache
}

// NewCachedUserRepository creates a cached user repository.
// This demonstrates cross-module dependencies:
// - UserRepository comes from the database module
// - Cache comes from the cache module
func NewCachedUserRepository(repo *UserRepository, cache *Cache) *CachedUserRepository {
	return &CachedUserRepository{
		repo:  repo,
		cache: cache,
	}
}

// FindAllCached returns users with caching.
func (r *CachedUserRepository) FindAllCached() ([]string, error) {
	// Check cache first (simplified - in real code, serialize properly)
	if cached, ok := r.cache.Get("users:all"); ok {
		return []string{cached}, nil
	}

	users, err := r.repo.FindAll()
	if err != nil {
		return nil, err
	}

	// Cache the result
	if len(users) > 0 {
		r.cache.Set("users:all", users[0])
	}

	return users, nil
}

// --- Application Service ---

// Application orchestrates the business logic.
type Application struct {
	cachedRepo *CachedUserRepository
}

// NewApplication creates the main application.
func NewApplication(cachedRepo *CachedUserRepository) *Application {
	return &Application{cachedRepo: cachedRepo}
}

// Run executes the application logic.
func (a *Application) Run() error {
	users, err := a.cachedRepo.FindAllCached()
	if err != nil {
		return err
	}

	fmt.Println("Users from database:")
	for _, user := range users {
		fmt.Printf("  - %s\n", user)
	}

	return nil
}

func run() error {
	app := gaz.New()

	// Register database module
	// Groups all database-related providers under a named module
	app.Module("database",
		func(c *gaz.Container) error {
			return gaz.For[*Database](c).
				Provider(func(_ *gaz.Container) (*Database, error) {
					return NewDatabase(), nil
				})
		},
		func(c *gaz.Container) error {
			return gaz.For[*UserRepository](c).
				Provider(func(c *gaz.Container) (*UserRepository, error) {
					db, err := gaz.Resolve[*Database](c)
					if err != nil {
						return nil, err
					}
					return NewUserRepository(db), nil
				})
		},
	)

	// Register cache module
	app.Module("cache",
		func(c *gaz.Container) error {
			return gaz.For[*Cache](c).
				Provider(func(_ *gaz.Container) (*Cache, error) {
					return NewCache(), nil
				})
		},
	)

	// Register service that depends on both modules
	app.Module("services",
		func(c *gaz.Container) error {
			return gaz.For[*CachedUserRepository](c).
				Provider(func(c *gaz.Container) (*CachedUserRepository, error) {
					repo, err := gaz.Resolve[*UserRepository](c)
					if err != nil {
						return nil, err
					}
					cache, err := gaz.Resolve[*Cache](c)
					if err != nil {
						return nil, err
					}
					return NewCachedUserRepository(repo, cache), nil
				})
		},
		func(c *gaz.Container) error {
			return gaz.For[*Application](c).
				Provider(func(c *gaz.Container) (*Application, error) {
					cachedRepo, err := gaz.Resolve[*CachedUserRepository](c)
					if err != nil {
						return nil, err
					}
					return NewApplication(cachedRepo), nil
				})
		},
	)

	// Build the container
	if err := app.Build(); err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}

	// Resolve and run the application
	application, err := gaz.Resolve[*Application](app.Container())
	if err != nil {
		return fmt.Errorf("failed to resolve Application: %w", err)
	}

	if err := application.Run(); err != nil {
		return fmt.Errorf("application error: %w", err)
	}

	// Note: In this example we don't call app.Run() because there's no
	// long-running service. For servers/workers, use app.Run(ctx) instead.
	return app.Stop(context.Background())
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
