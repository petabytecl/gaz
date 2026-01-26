package gaz

// Defaulter allows a config struct to set its own default values.
// The Default() method is called after unmarshaling but before validation.
type Defaulter interface {
	Default()
}

// Validator allows a config struct to validate its own state.
// The Validate() method is called after defaults are applied.
// If it returns an error, the application startup will fail.
type Validator interface {
	Validate() error
}
