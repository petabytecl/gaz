package di

import (
	"testing"
)

// BenchmarkResolveSingleton benchmarks resolving a singleton service.
func BenchmarkResolveSingleton(b *testing.B) {
	c := New()
	For[*benchTestService](c).Provider(func(*Container) (*benchTestService, error) {
		return &benchTestService{}, nil
	})
	if err := c.Build(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Resolve[*benchTestService](c)
	}
}

// BenchmarkResolveTransient benchmarks resolving a transient service.
func BenchmarkResolveTransient(b *testing.B) {
	c := New()
	For[*benchTestService](c).Transient().Provider(func(*Container) (*benchTestService, error) {
		return &benchTestService{}, nil
	})
	if err := c.Build(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Resolve[*benchTestService](c)
	}
}

// BenchmarkResolveWithDependencies benchmarks resolving a service with dependencies.
func BenchmarkResolveWithDependencies(b *testing.B) {
	c := New()
	For[*benchTestDep](c).Provider(func(*Container) (*benchTestDep, error) {
		return &benchTestDep{}, nil
	})
	For[*benchTestService](c).Provider(func(c *Container) (*benchTestService, error) {
		_, err := Resolve[*benchTestDep](c)
		if err != nil {
			return nil, err
		}
		return &benchTestService{}, nil
	})
	if err := c.Build(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Resolve[*benchTestService](c)
	}
}

// BenchmarkRegister benchmarks service registration.
func BenchmarkRegister(b *testing.B) {
	c := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = For[*testService](c).Provider(func(*Container) (*testService, error) {
			return &testService{}, nil
		})
	}
}

type benchTestService struct{}

type benchTestDep struct{}
