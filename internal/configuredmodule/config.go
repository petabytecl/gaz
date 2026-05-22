// Package configuredmodule centralizes repeated module config provider wiring.
package configuredmodule

import (
	"fmt"

	"github.com/petabytecl/gaz"
)

type namespacedValidatedConfig[T any] interface {
	*T
	Namespace() string
	Validate() error
}

type defaultedConfig[T any] interface {
	namespacedValidatedConfig[T]
	SetDefaults()
}

// ProvideValidated registers a config provider that starts from defaults,
// optionally overlays ProviderValues, and validates the resulting config.
func ProvideValidated[T any, PT namespacedValidatedConfig[T]](
	defaults *T,
	registrationName string,
	validationContext string,
) func(*gaz.Container) error {
	return provide(defaults, registrationName, validationContext, func(PT) {})
}

// ProvideDefaulted registers a config provider that starts from defaults,
// optionally overlays ProviderValues, reapplies zero-value defaults, and
// validates the resulting config.
func ProvideDefaulted[T any, PT defaultedConfig[T]](
	defaults *T,
	registrationName string,
	validationContext string,
) func(*gaz.Container) error {
	return provide(defaults, registrationName, validationContext, func(cfg PT) {
		cfg.SetDefaults()
	})
}

func provide[T any, PT namespacedValidatedConfig[T]](
	defaults *T,
	registrationName string,
	validationContext string,
	beforeValidate func(PT),
) func(*gaz.Container) error {
	return func(c *gaz.Container) error {
		if err := gaz.For[T](c).Provider(func(c *gaz.Container) (T, error) {
			cfg := *defaults
			cfgPtr := PT(&cfg)

			if pv, err := gaz.Resolve[*gaz.ProviderValues](c); err == nil {
				if unmarshalErr := pv.UnmarshalKey(cfgPtr.Namespace(), &cfg); unmarshalErr != nil {
					_ = unmarshalErr
				}
			}

			beforeValidate(cfgPtr)
			if err := cfgPtr.Validate(); err != nil {
				return cfg, fmt.Errorf("%s: %w", validationContext, err)
			}

			return cfg, nil
		}); err != nil {
			return fmt.Errorf("register %s: %w", registrationName, err)
		}
		return nil
	}
}
