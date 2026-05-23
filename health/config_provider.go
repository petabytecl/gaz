package health

// ModuleName is the canonical identity for the health module.
// Both auto-registration and explicit health/module.New() use this name
// so duplicate detection works correctly.
const ModuleName = "health"

// HealthConfigProvider is implemented by config structs that provide health settings.
// When a config implementing this interface is passed to service.Builder,
// the health module is automatically registered.
//
// Example:
//
//	type AppConfig struct {
//	    Port   int
//	    Health health.Config
//	}
//
//	func (c *AppConfig) HealthConfig() health.Config {
//	    return c.Health
//	}
//
//	// Health module auto-registers because AppConfig implements HealthConfigProvider
//	app, _ := service.New().
//	    WithConfig(&AppConfig{Health: health.DefaultConfig()}).
//	    Build()
type HealthConfigProvider interface {
	HealthConfig() Config
}
