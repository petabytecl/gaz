package di_test

import (
	"testing"

	"github.com/petabytecl/gaz/di"
)

// sink prevents compiler optimisation of benchmark results.
//
//nolint:gochecknoglobals // required for benchmark correctness
var sink any

// benchService is a minimal struct used in benchmarks.
type benchService struct {
	Value int
}

// benchInterface is an interface for ResolveAll benchmarks.
type benchInterface interface {
	ID() int
}

type benchImpl struct{ id int }

func (b *benchImpl) ID() int { return b.id }

func BenchmarkResolve_Singleton(b *testing.B) {
	b.ReportAllocs()

	c := di.New()
	_ = di.For[*benchService](c).ProviderFunc(func(_ *di.Container) *benchService {
		return &benchService{Value: 42}
	})
	// Warm up singleton.
	_, _ = di.Resolve[*benchService](c)

	for b.Loop() {
		v, _ := di.Resolve[*benchService](c)
		sink = v
	}
}

func BenchmarkResolve_Transient(b *testing.B) {
	b.ReportAllocs()

	c := di.New()
	_ = di.For[*benchService](c).Transient().ProviderFunc(func(_ *di.Container) *benchService {
		return &benchService{Value: 42}
	})

	for b.Loop() {
		v, _ := di.Resolve[*benchService](c)
		sink = v
	}
}

func BenchmarkResolve_Named(b *testing.B) {
	b.ReportAllocs()

	c := di.New()
	_ = di.For[*benchService](c).Named("primary").ProviderFunc(func(_ *di.Container) *benchService {
		return &benchService{Value: 1}
	})
	// Warm up singleton.
	_, _ = di.Resolve[*benchService](c, di.Named("primary"))

	for b.Loop() {
		v, _ := di.Resolve[*benchService](c, di.Named("primary"))
		sink = v
	}
}

func BenchmarkResolve_Parallel(b *testing.B) {
	b.ReportAllocs()

	c := di.New()
	_ = di.For[*benchService](c).ProviderFunc(func(_ *di.Container) *benchService {
		return &benchService{Value: 42}
	})
	// Warm up singleton.
	_, _ = di.Resolve[*benchService](c)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			v, _ := di.Resolve[*benchService](c)
			sink = v
		}
	})
}

func BenchmarkResolveAll(b *testing.B) {
	b.ReportAllocs()

	c := di.New()
	for i := range 10 {
		id := i
		_ = di.For[*benchImpl](c).ProviderFunc(func(_ *di.Container) *benchImpl {
			return &benchImpl{id: id}
		})
	}

	for b.Loop() {
		v, _ := di.ResolveAll[benchInterface](c)
		sink = v
	}
}
