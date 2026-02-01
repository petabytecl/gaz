package gaztest

import (
	"github.com/petabytecl/gaz"
)

// RequireResolve resolves type T from the test app and fails the test if resolution fails.
// This is a convenience wrapper that calls gaz.Resolve[T] and fails with t.Fatalf on error.
//
// Example:
//
//	db := gaztest.RequireResolve[*Database](t, app)
//	// use db directly - no error check needed
func RequireResolve[T any](tb TB, app *App) T {
	tb.Helper()
	result, err := gaz.Resolve[T](app.Container())
	if err != nil {
		tb.Fatalf("gaztest: RequireResolve[%s]: %v", gaz.TypeName[T](), err)
	}
	return result
}
