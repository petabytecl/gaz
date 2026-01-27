# Modules Example

This example demonstrates organizing providers into modules for better code organization.

## What It Demonstrates

1. **Module Organization**: Group related providers under named modules
2. **Cross-Module Dependencies**: Services can depend on providers from other modules
3. **Clean Separation**: Each module encapsulates its domain (database, cache, services)
4. **Duplicate Detection**: Module names must be unique (ErrDuplicateModule if repeated)

## Running

```bash
cd examples/modules
go run .
```

Output:
```
Users from database:
  - row1
  - row2
  - row3
```

## Module Structure

```
modules/
├── database module
│   ├── *Database        (database connection)
│   └── *UserRepository  (data access layer)
├── cache module
│   └── *Cache           (in-memory cache)
└── services module
    ├── *CachedUserRepository  (depends on both modules)
    └── *Application           (main entry point)
```

## Key Patterns

### Creating a Module

```go
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
```

### Cross-Module Dependencies

Services in one module can depend on services from other modules:

```go
app.Module("services",
    func(c *gaz.Container) error {
        return gaz.For[*CachedUserRepository](c).
            Provider(func(c *gaz.Container) (*CachedUserRepository, error) {
                // From database module
                repo, err := gaz.Resolve[*UserRepository](c)
                if err != nil {
                    return nil, err
                }
                // From cache module
                cache, err := gaz.Resolve[*Cache](c)
                if err != nil {
                    return nil, err
                }
                return NewCachedUserRepository(repo, cache), nil
            })
    },
)
```

## When to Use Modules

**Use modules when:**
- You have groups of related services (database layer, HTTP layer, etc.)
- You want to encapsulate domain boundaries
- You're building a larger application with multiple teams/concerns

**Use direct registration when:**
- You have a small application with few services
- Services don't naturally group together
- You prefer explicit registration over organization

## Module Benefits

1. **Organization**: Related providers grouped together
2. **Documentation**: Module names describe purpose
3. **Error Context**: Errors include module name for debugging
4. **Duplicate Prevention**: Module names must be unique
