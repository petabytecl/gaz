package health

import (
	"github.com/petabytecl/gaz/health/internal/healthx"
)

// IETFResultWriter delegates to healthx.IETFResultWriter.
// Kept for backward compatibility with existing code that references health.IETFResultWriter.
type IETFResultWriter = healthx.IETFResultWriter

// NewIETFResultWriter creates a new IETFResultWriter.
// This is an alias to healthx.NewIETFResultWriter for backward compatibility.
func NewIETFResultWriter(opts ...healthx.IETFWriterOption) *healthx.IETFResultWriter {
	return healthx.NewIETFResultWriter(opts...)
}
