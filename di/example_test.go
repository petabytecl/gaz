package di_test

import (
	"fmt"
	"strings"

	"github.com/petabytecl/gaz/di"
)

// Service types used in examples

// Database represents a database connection.
type Database struct {
	Host string
}

// UserRepository depends on Database.
type UserRepository struct {
	DB *Database
}

// Config is a simple configuration struct.
type Config struct {
	Debug bool
}

// Counter tracks resolution calls for singleton vs transient demo.
type Counter struct {
	value int
}

func (c *Counter) Next() int {
	c.value++
	return c.value
}

// =============================================================================
// Container Examples
// =============================================================================

// ExampleNew demonstrates creating a new DI container.
// The container starts empty and services are registered using For[T]().
func ExampleNew() {
	c := di.New()

	// Register a service
	err := di.For[*Database](c).Provider(func(_ *di.Container) (*Database, error) {
		return &Database{Host: "localhost"}, nil
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Build the container (validates registrations)
	if err := c.Build(); err != nil {
		fmt.Println("error:", err)
		return
	}

	// List registered services
	services := c.List()
	fmt.Println("registered:", len(services), "service(s)")
	// Output: registered: 1 service(s)
}

// ExampleContainer_Build demonstrates the container Build process.
// Eager services are instantiated during Build, while lazy services
// are instantiated on first resolution.
func ExampleContainer_Build() {
	c := di.New()

	instantiated := false
	err := di.For[*Database](c).Eager().Provider(func(_ *di.Container) (*Database, error) {
		instantiated = true
		return &Database{Host: "localhost"}, nil
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

// ExampleContainer_List demonstrates listing registered services.
// Service names are returned in sorted order.
func ExampleContainer_List() {
	c := di.New()

	di.For[*Config](c).Instance(&Config{Debug: true})
	di.For[*Database](c).Instance(&Database{Host: "localhost"})

	services := c.List()
	fmt.Println("registered services:", len(services))
	// Output: registered services: 2
}

// =============================================================================
// For[T]() Registration Examples
// =============================================================================

// ExampleFor_singleton demonstrates the default singleton scope.
// Singletons return the same instance on every resolution.
func ExampleFor_singleton() {
	c := di.New()

	counter := &Counter{}
	err := di.For[*Counter](c).Instance(counter)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Resolve twice - same instance returned
	c1, _ := di.Resolve[*Counter](c)
	c2, _ := di.Resolve[*Counter](c)

	fmt.Println("same instance:", c1 == c2)
	// Output: same instance: true
}

// ExampleFor_transient demonstrates transient scope.
// Transient services create a new instance on every resolution.
func ExampleFor_transient() {
	c := di.New()

	callCount := 0
	err := di.For[*Counter](c).Transient().Provider(func(_ *di.Container) (*Counter, error) {
		callCount++
		return &Counter{value: callCount * 100}, nil
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Resolve twice - different instances created
	c1, _ := di.Resolve[*Counter](c)
	c2, _ := di.Resolve[*Counter](c)

	fmt.Println("different instances:", c1 != c2)
	fmt.Println("c1 value:", c1.value)
	fmt.Println("c2 value:", c2.value)
	// Output:
	// different instances: true
	// c1 value: 100
	// c2 value: 200
}

// ExampleFor_eager demonstrates eager instantiation.
// Eager services are created when Build() is called, not on first resolution.
func ExampleFor_eager() {
	c := di.New()

	var buildOrder []string
	err := di.For[*Database](c).Eager().Provider(func(_ *di.Container) (*Database, error) {
		buildOrder = append(buildOrder, "Database created")
		return &Database{Host: "localhost"}, nil
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("before build: created =", len(buildOrder))

	if err := c.Build(); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("after build: created =", len(buildOrder))
	// Output:
	// before build: created = 0
	// after build: created = 1
}

// ExampleFor_instance demonstrates registering a pre-built instance.
// No provider function is needed - the value is returned directly.
func ExampleFor_instance() {
	c := di.New()

	// Register a pre-built configuration
	cfg := &Config{Debug: true}
	err := di.For[*Config](c).Instance(cfg)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Resolve returns the same instance
	resolved, err := di.Resolve[*Config](c)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("same instance:", resolved == cfg)
	fmt.Println("debug:", resolved.Debug)
	// Output:
	// same instance: true
	// debug: true
}

// ExampleFor_named demonstrates named registrations.
// Multiple services of the same type can be registered with different names.
func ExampleFor_named() {
	c := di.New()

	// Register two databases with different names
	err := di.For[*Database](c).Named("primary").Instance(&Database{Host: "primary.db"})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	err = di.For[*Database](c).Named("replica").Instance(&Database{Host: "replica.db"})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Resolve by name
	primary, err := di.Resolve[*Database](c, di.Named("primary"))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	replica, err := di.Resolve[*Database](c, di.Named("replica"))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("primary:", primary.Host)
	fmt.Println("replica:", replica.Host)
	// Output:
	// primary: primary.db
	// replica: replica.db
}

// =============================================================================
// Resolve Examples
// =============================================================================

// ExampleResolve demonstrates basic service resolution.
func ExampleResolve() {
	c := di.New()

	err := di.For[*Database](c).Instance(&Database{Host: "localhost:5432"})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	db, err := di.Resolve[*Database](c)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("host:", db.Host)
	// Output: host: localhost:5432
}

// ExampleResolve_withDependencies demonstrates automatic dependency wiring.
// When a provider resolves its dependencies, the container automatically
// injects them.
func ExampleResolve_withDependencies() {
	c := di.New()

	// Register Database (no dependencies)
	err := di.For[*Database](c).Provider(func(_ *di.Container) (*Database, error) {
		return &Database{Host: "localhost"}, nil
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Register UserRepository (depends on Database)
	err = di.For[*UserRepository](c).Provider(func(c *di.Container) (*UserRepository, error) {
		db, err := di.Resolve[*Database](c)
		if err != nil {
			return nil, err
		}
		return &UserRepository{DB: db}, nil
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Build container
	if err := c.Build(); err != nil {
		fmt.Println("error:", err)
		return
	}

	// Resolve UserRepository - Database is automatically injected
	repo, err := di.Resolve[*UserRepository](c)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("repository has database:", repo.DB != nil)
	fmt.Println("database host:", repo.DB.Host)
	// Output:
	// repository has database: true
	// database host: localhost
}

// ExampleMustResolve demonstrates panic-on-error resolution.
// Use only in main() or test setup where failure is fatal.
func ExampleMustResolve() {
	c := di.New()

	di.For[*Config](c).Instance(&Config{Debug: true})

	// MustResolve panics if service not found
	cfg := di.MustResolve[*Config](c)

	fmt.Println("debug:", cfg.Debug)
	// Output: debug: true
}

// =============================================================================
// Module Examples
// =============================================================================

// ExampleNewModuleFunc demonstrates creating a module with NewModuleFunc.
// Modules bundle related service registrations together.
func ExampleNewModuleFunc() {
	c := di.New()

	// Create a module that registers database infrastructure
	dbModule := di.NewModuleFunc("database", func(c *di.Container) error {
		err := di.For[*Database](c).Provider(func(_ *di.Container) (*Database, error) {
			return &Database{Host: "localhost:5432"}, nil
		})
		if err != nil {
			return err
		}

		return di.For[*UserRepository](c).Provider(func(c *di.Container) (*UserRepository, error) {
			db, err := di.Resolve[*Database](c)
			if err != nil {
				return nil, err
			}
			return &UserRepository{DB: db}, nil
		})
	})

	// Register the module
	if err := dbModule.Register(c); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("module:", dbModule.Name())

	// Build and resolve
	if err := c.Build(); err != nil {
		fmt.Println("error:", err)
		return
	}

	repo, _ := di.Resolve[*UserRepository](c)
	fmt.Println("repository created:", repo != nil)
	// Output:
	// module: database
	// repository created: true
}

// ExampleModule demonstrates implementing the Module interface.
// Custom modules can encapsulate complex registration logic.
func ExampleModule() {
	c := di.New()

	// InfraModule implements di.Module interface
	module := &InfraModule{host: "db.example.com"}

	// Register the module
	if err := module.Register(c); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("module:", module.Name())

	// Build and resolve
	if err := c.Build(); err != nil {
		fmt.Println("error:", err)
		return
	}

	db, _ := di.Resolve[*Database](c)
	fmt.Println("database host:", db.Host)
	// Output:
	// module: infra
	// database host: db.example.com
}

// InfraModule implements di.Module interface.
type InfraModule struct {
	host string
}

func (m *InfraModule) Name() string {
	return "infra"
}

func (m *InfraModule) Register(c *di.Container) error {
	return di.For[*Database](c).Provider(func(_ *di.Container) (*Database, error) {
		return &Database{Host: m.host}, nil
	})
}

// =============================================================================
// Utility Examples
// =============================================================================

// ExampleTypeName demonstrates getting the type name for a type.
// Type names are used as service registration keys by default.
func ExampleTypeName() {
	// Type names include the full package path
	dbType := di.TypeName[*Database]()
	cfgType := di.TypeName[*Config]()

	// Check that type names contain expected substrings
	hasDatabase := strings.Contains(dbType, "Database")
	hasConfig := strings.Contains(cfgType, "Config")

	fmt.Println("Database type contains 'Database':", hasDatabase)
	fmt.Println("Config type contains 'Config':", hasConfig)
	// Output:
	// Database type contains 'Database': true
	// Config type contains 'Config': true
}

// ExampleHas demonstrates checking if a service is registered.
func ExampleHas() {
	c := di.New()

	di.For[*Config](c).Instance(&Config{})

	fmt.Println("Config registered:", di.Has[*Config](c))
	fmt.Println("Database registered:", di.Has[*Database](c))
	// Output:
	// Config registered: true
	// Database registered: false
}
