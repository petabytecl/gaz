package backoff_test

import (
	"testing"
	"time"

	"github.com/petabytecl/gaz/backoff"
)

// sink prevents compiler optimisation of benchmark results.
//
//nolint:gochecknoglobals // required for benchmark correctness
var sink time.Duration

func BenchmarkExponentialBackOff_NextBackOff(b *testing.B) {
	b.ReportAllocs()

	bo := backoff.NewExponentialBackOff()

	for b.Loop() {
		d := bo.NextBackOff()
		if d == backoff.Stop {
			bo.Reset()
		}
		sink = d
	}
}

func BenchmarkConstantBackOff_NextBackOff(b *testing.B) {
	b.ReportAllocs()

	bo := backoff.NewConstantBackOff(time.Second)

	for b.Loop() {
		sink = bo.NextBackOff()
	}
}

func BenchmarkExponentialBackOff_Reset(b *testing.B) {
	b.ReportAllocs()

	bo := backoff.NewExponentialBackOff()

	for b.Loop() {
		bo.Reset()
	}
}
