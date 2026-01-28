package gaz

import "github.com/petabytecl/gaz/config"

// Defaulter allows a config struct to set its own default values.
// The Default() method is called after unmarshaling but before validation.
// Deprecated: Import github.com/petabytecl/gaz/config directly.
type Defaulter = config.Defaulter

// Validator allows a config struct to validate its own state.
// The Validate() method is called after defaults are applied.
// If it returns an error, the application startup will fail.
// Deprecated: Import github.com/petabytecl/gaz/config directly.
type Validator = config.Validator
