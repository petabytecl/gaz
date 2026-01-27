# Cobra CLI Example

This example demonstrates integrating gaz with Cobra CLI for command-line applications.

## What It Demonstrates

1. **Root Command with Persistent Flags**: Flags available to all subcommands
2. **Subcommands with DI**: Commands that access injected services
3. **Flag to Config Binding**: Cobra flags populate configuration structs
4. **WithCobra() Integration**: Automatic lifecycle management

## Running

```bash
cd examples/cobra-cli
go build -o myapp .

# Show help
./myapp --help

# Start server with default settings
./myapp serve

# Start with custom port and debug mode
./myapp serve --port 9000 --debug

# Version info (simple command, no DI)
./myapp version
```

## Available Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--debug` | `-d` | false | Enable debug mode |
| `--port` | `-p` | 8080 | Server port |
| `--host` | `-H` | localhost | Server host |
| `--timeout` | `-t` | 30 | Request timeout in seconds |

## Available Commands

| Command | Description |
|---------|-------------|
| `serve` | Start the server |
| `version` | Print version information |

## Example Session

```bash
$ ./myapp serve --port 9000 --debug
Initializing server on localhost:9000...
Server starting on localhost:9000
Debug mode: true
Request timeout: 30s
Server is running... (Ctrl+C to stop)
^C
Server shutting down...
Cleaning up server resources...
```

## Key Patterns

### Persistent Flags

```go
rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
rootCmd.PersistentFlags().IntP("port", "p", 8080, "Server port")
```

Persistent flags are inherited by all subcommands.

### Reading Flags in Handler

```go
func runServe(cmd *cobra.Command, _ []string) error {
    debug, _ := cmd.Flags().GetBool("debug")
    port, _ := cmd.Flags().GetInt("port")
    
    config := AppConfig{
        Debug: debug,
        Port:  port,
    }
    
    app := gaz.New()
    app.ProvideInstance(config)
    // ...
}
```

### WithCobra() Integration

```go
app.WithCobra(cmd)
```

This hooks into:
- `PersistentPreRunE`: Calls `Build()` and `Start()` 
- `PersistentPostRunE`: Calls `Stop()` for cleanup

### Accessing DI from Context

```go
// In any subcommand handler
app := gaz.FromContext(cmd.Context())
server, _ := gaz.Resolve[*Server](app.Container())
```

## Architecture

```
myapp
├── rootCmd (persistent flags)
│   ├── serve   (uses DI, runs server)
│   └── version (simple, no DI)
```

The `serve` command:
1. Creates gaz app with configuration from flags
2. Registers Server with lifecycle hooks
3. Uses WithCobra() for automatic lifecycle management
4. Runs until interrupted
