// Package configuredmodule centralizes repeated module config provider wiring.
package configuredmodule

import (
	"errors"
	"fmt"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/config"
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
	return provide(defaults, registrationName, validationContext, func(PT) error {
		return nil
	})
}

// ProvideDefaulted registers a config provider that starts from defaults,
// optionally overlays ProviderValues, reapplies zero-value defaults, and
// validates the resulting config.
func ProvideDefaulted[T any, PT defaultedConfig[T]](
	defaults *T,
	registrationName string,
	validationContext string,
) func(*gaz.Container) error {
	return ProvideDefaultedWithHook[T, PT](defaults, registrationName, validationContext, nil)
}

// ProvideDefaultedWithHook registers a config provider like ProvideDefaulted,
// then runs a hook before validation. This keeps uncommon module-specific
// fallback policy behind the same configured module registration rule.
func ProvideDefaultedWithHook[T any, PT defaultedConfig[T]](
	defaults *T,
	registrationName string,
	validationContext string,
	beforeValidate func(PT) error,
) func(*gaz.Container) error {
	return provide(defaults, registrationName, validationContext, func(cfg PT) error {
		cfg.SetDefaults()
		if beforeValidate == nil {
			return nil
		}
		return beforeValidate(cfg)
	})
}

func provide[T any, PT namespacedValidatedConfig[T]](
	defaults *T,
	registrationName string,
	validationContext string,
	beforeValidate func(PT) error,
) func(*gaz.Container) error {
	return func(c *gaz.Container) error {
		if defaults == nil {
			return fmt.Errorf("register %s: defaults must not be nil", registrationName)
		}

		if err := gaz.For[T](c).Provider(func(c *gaz.Container) (T, error) {
			cfg := *defaults
			cfgPtr := PT(&cfg)

			if err := OverlayProviderValues(c, cfgPtr); err != nil {
				return cfg, err
			}

			if err := beforeValidate(cfgPtr); err != nil {
				return cfg, fmt.Errorf("%s: %w", validationContext, err)
			}

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

// OverlayProviderValues overlays a module-owned config from ProviderValues.
// A missing ProviderValues service or missing namespace keeps defaults. Other
// errors are malformed config and should fail at the config module seam.
func OverlayProviderValues(c *gaz.Container, cfg interface{ Namespace() string }) error {
	pv, err := gaz.Resolve[*gaz.ProviderValues](c)
	if err != nil {
		if errors.Is(err, gaz.ErrDINotFound) {
			return nil
		}
		return fmt.Errorf("resolve ProviderValues: %w", err)
	}

	if unmarshalErr := pv.UnmarshalKey(cfg.Namespace(), cfg); unmarshalErr != nil {
		if errors.Is(unmarshalErr, config.ErrKeyNotFound) {
			return nil
		}
		return fmt.Errorf("load provider config %q: %w", cfg.Namespace(), unmarshalErr)
	}

	return nil
}
