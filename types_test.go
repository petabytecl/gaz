package gaz

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// TypesSuite tests type-related utilities.
type TypesSuite struct {
	suite.Suite
}

func TestTypesSuite(t *testing.T) {
	suite.Run(t, new(TypesSuite))
}

func (s *TypesSuite) TestTypeName() {
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
		s.Run(tt.name, func() {
			s.Equal(tt.expected, tt.got)
		})
	}
}

func (s *TypesSuite) TestTypeNamePointer() {
	got := TypeName[*string]()
	expected := "*string"
	s.Equal(expected, got)
}

func (s *TypesSuite) TestTypeNameSlice() {
	got := TypeName[[]string]()
	expected := "[]string"
	s.Equal(expected, got)
}

func (s *TypesSuite) TestTypeNameMap() {
	got := TypeName[map[string]int]()
	expected := "map[string]int"
	s.Equal(expected, got)
}
