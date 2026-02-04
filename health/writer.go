package health

import (
	"github.com/petabytecl/gaz/health/internal"
)

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
