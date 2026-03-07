package config_test

import (
	"testing"

	"github.com/petabytecl/gaz/config"
)

func BenchmarkFieldError_String(b *testing.B) {
	feWithTag := config.NewFieldError("Config.database.host", "required", "", "required field cannot be empty")
	feWithoutTag := config.FieldError{
		Namespace: "Config.database.host",
		Message:   "custom error",
	}

	b.Run("WithTag", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = feWithTag.String()
		}
	})

	b.Run("WithoutTag", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = feWithoutTag.String()
		}
	})
}
