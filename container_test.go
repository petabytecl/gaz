package gaz

import "testing"

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
	if c.services == nil {
		t.Fatal("New() did not initialize services map")
	}
	if c.built {
		t.Fatal("New container should not be built")
	}
}

func TestNewReturnsDistinctInstances(t *testing.T) {
	c1 := New()
	c2 := New()
	if c1 == c2 {
		t.Fatal("New() should return distinct instances")
	}
}
