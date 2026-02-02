package health

import (
	"github.com/petabytecl/gaz/health/internal"
)

// Checker is the interface for executing health checks.
// This is a type alias to allow external consumers to use health.Checker
// instead of importing the internal package.
type Checker = internal.Checker

// CheckerOption configures the Checker.
// This is a type alias to allow external consumers to use health.CheckerOption
// instead of importing the internal package.
type CheckerOption = internal.CheckerOption

// IETFWriterOption configures the IETFResultWriter.
// This is a type alias to allow external consumers to use health.IETFWriterOption
// instead of importing the internal package.
type IETFWriterOption = internal.IETFWriterOption

// IETFResultWriter delegates to internal.IETFResultWriter.
// Kept for backward compatibility with existing code that references health.IETFResultWriter.
type IETFResultWriter = internal.IETFResultWriter

// NewIETFResultWriter creates a new IETFResultWriter.
// This is an alias to internal.NewIETFResultWriter for backward compatibility.
func NewIETFResultWriter(opts ...IETFWriterOption) *IETFResultWriter {
	return internal.NewIETFResultWriter(opts...)
}
