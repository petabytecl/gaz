package config_test

import (
	"testing"

	"github.com/petabytecl/gaz/config"
)

func BenchmarkFieldError_String(b *testing.B) {
	b.ReportAllocs()
	feWithTag := config.NewFieldError("Config.database.host", "required", "", "required field cannot be empty")
	feWithoutTag := config.FieldError{
		Namespace: "Config.database.host",
		Message:   "custom error",
	}

	b.Run("WithTag", func(b *testing.B) {
		for b.Loop() {
			_ = feWithTag.String()
		}
	})

	b.Run("WithoutTag", func(b *testing.B) {
		for b.Loop() {
			_ = feWithoutTag.String()
		}
	})
}
