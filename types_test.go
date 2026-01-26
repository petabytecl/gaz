package gaz

import "testing"

func TestTypeName(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"string", TypeName[string](), "string"},
		{"int", TypeName[int](), "int"},
		{"bool", TypeName[bool](), "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("TypeName[%s]() = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestTypeNamePointer(t *testing.T) {
	got := TypeName[*string]()
	expected := "*string"
	if got != expected {
		t.Errorf("TypeName[*string]() = %q, want %q", got, expected)
	}
}

func TestTypeNameSlice(t *testing.T) {
	got := TypeName[[]string]()
	expected := "[]string"
	if got != expected {
		t.Errorf("TypeName[[]string]() = %q, want %q", got, expected)
	}
}

func TestTypeNameMap(t *testing.T) {
	got := TypeName[map[string]int]()
	expected := "map[string]int"
	if got != expected {
		t.Errorf("TypeName[map[string]int]() = %q, want %q", got, expected)
	}
}
