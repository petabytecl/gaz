package config

// Defaulter allows a config struct to set its own default values.
// The Default() method is called after unmarshaling but before validation.
//
// Example:
//
//	type AppConfig struct {
//	    Port int `mapstructure:"port"`
//	}
//
//	func (c *AppConfig) Default() {
//	    if c.Port == 0 {
//	        c.Port = 8080
//	    }
//	}
type Defaulter interface {
	Default()
}

// Validator allows a config struct to validate its own state.
// The Validate() method is called after defaults are applied.
// If it returns an error, the configuration loading will fail.
//
// This is for custom validation logic. For struct tag validation
// (e.g., `validate:"required"`), use go-playground/validator tags
// which are automatically validated during LoadInto.
//
// Example:
//
//	type AppConfig struct {
//	    Host string `mapstructure:"host"`
//	    Port int    `mapstructure:"port"`
//	}
//
//	func (c *AppConfig) Validate() error {
//	    if c.Host == "" && c.Port == 0 {
//	        return fmt.Errorf("either host or port must be set")
//	    }
//	    return nil
//	}
type Validator interface {
	Validate() error
}
