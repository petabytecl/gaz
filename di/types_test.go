package di

import (
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
)

// =============================================================================
// TypesSuite - Tests for TypeNameReflect and typeName
// =============================================================================

type TypesSuite struct {
	suite.Suite
}

func TestTypesSuite(t *testing.T) {
	suite.Run(t, new(TypesSuite))
}

// =============================================================================
// TypeNameReflect Tests
// =============================================================================

func (s *TypesSuite) TestTypeNameReflect_WithReflectType() {
	// Pass reflect.Type directly
	typ := reflect.TypeOf((*testTypesStruct)(nil)).Elem()
	name := TypeNameReflect(typ)
	s.Contains(name, "testTypesStruct", "should use typeName for reflect.Type")
}

func (s *TypesSuite) TestTypeNameReflect_WithPointerReflectType() {
	// Pass reflect.Type of a pointer
	typ := reflect.TypeOf((*testTypesStruct)(nil))
	name := TypeNameReflect(typ)
	s.Contains(name, "*")
	s.Contains(name, "testTypesStruct", "should handle pointer reflect.Type")
}

func (s *TypesSuite) TestTypeNameReflect_WithRegularValue() {
	// Pass regular value (not reflect.Type)
	val := testTypesStruct{value: "test"}
	name := TypeNameReflect(val)
	s.Contains(name, "testTypesStruct", "should call reflect.TypeOf first")
}

func (s *TypesSuite) TestTypeNameReflect_WithPointerValue() {
	// Pass pointer value
	val := &testTypesStruct{value: "test"}
	name := TypeNameReflect(val)
	s.Contains(name, "*")
	s.Contains(name, "testTypesStruct", "should handle pointer value")
}

func (s *TypesSuite) TestTypeNameReflect_WithBuiltinType() {
	name := TypeNameReflect(42)
	s.Equal("int", name, "should return builtin type name without package")
}

func (s *TypesSuite) TestTypeNameReflect_WithString() {
	name := TypeNameReflect("hello")
	s.Equal("string", name, "should return string type name")
}

// =============================================================================
// typeName Edge Cases Tests
// =============================================================================

func (s *TypesSuite) TestTypeName_Nil() {
	name := typeName(nil)
	s.Equal("nil", name, "nil type should return 'nil'")
}

func (s *TypesSuite) TestTypeName_NamedTypeWithPackagePath() {
	// A struct from this package has a package path
	typ := reflect.TypeOf(testTypesStruct{})
	name := typeName(typ)
	s.Contains(name, "github.com/petabytecl/gaz/di", "should include package path")
	s.Contains(name, "testTypesStruct", "should include type name")
}

func (s *TypesSuite) TestTypeName_NamedTypeWithoutPackagePath() {
	// Builtins like "int" have no package path
	typ := reflect.TypeOf(0)
	name := typeName(typ)
	s.Equal("int", name, "builtin should return just the name")
}

func (s *TypesSuite) TestTypeName_PointerType() {
	typ := reflect.TypeOf((*testTypesStruct)(nil))
	name := typeName(typ)
	s.True(name[0] == '*', "pointer type should start with *")
	s.Contains(name, "testTypesStruct", "should include element type")
}

func (s *TypesSuite) TestTypeName_SliceType() {
	typ := reflect.TypeOf([]testTypesStruct{})
	name := typeName(typ)
	s.True(len(name) >= 2 && name[0:2] == "[]", "slice type should start with []")
	s.Contains(name, "testTypesStruct", "should include element type")
}

func (s *TypesSuite) TestTypeName_MapType() {
	typ := reflect.TypeOf(map[string]testTypesStruct{})
	name := typeName(typ)
	s.Contains(name, "map[", "map type should contain map[")
	s.Contains(name, "string", "should include key type")
	s.Contains(name, "testTypesStruct", "should include value type")
}

func (s *TypesSuite) TestTypeName_InterfaceType() {
	// Get interface type using reflect
	typ := reflect.TypeOf((*io.Reader)(nil)).Elem()
	name := typeName(typ)
	// Interface is a named type with package path
	s.Contains(name, "io", "should include package")
	s.Contains(name, "Reader", "should include interface name")
}

func (s *TypesSuite) TestTypeName_EmptyInterface() {
	// Empty interface type (any/interface{})
	var val any
	typ := reflect.TypeOf(&val).Elem()
	name := typeName(typ)
	// Empty interface is unnamed, hits default case
	s.NotEmpty(name, "should return some name for empty interface")
}

func (s *TypesSuite) TestTypeName_SliceOfPointers() {
	typ := reflect.TypeOf([]*testTypesStruct{})
	name := typeName(typ)
	s.Contains(name, "[]", "should have slice prefix")
	s.Contains(name, "*", "should have pointer marker")
	s.Contains(name, "testTypesStruct", "should have type name")
}

func (s *TypesSuite) TestTypeName_MapWithPointerValue() {
	typ := reflect.TypeOf(map[string]*testTypesStruct{})
	name := typeName(typ)
	s.Contains(name, "map[", "should be a map")
	s.Contains(name, "*", "should have pointer in value")
	s.Contains(name, "testTypesStruct", "should have value type name")
}

func (s *TypesSuite) TestTypeName_PointerToPointer() {
	typ := reflect.TypeOf((**testTypesStruct)(nil))
	name := typeName(typ)
	s.True(len(name) >= 2 && name[0:2] == "**", "should have double pointer prefix")
}

func (s *TypesSuite) TestTypeName_SliceOfBuiltins() {
	typ := reflect.TypeOf([]int{})
	name := typeName(typ)
	s.Equal("[]int", name, "slice of builtin should be []int")
}

func (s *TypesSuite) TestTypeName_MapOfBuiltins() {
	typ := reflect.TypeOf(map[string]int{})
	name := typeName(typ)
	s.Equal("map[string]int", name, "map of builtins should be map[string]int")
}

// =============================================================================
// TypeName Generic Function Tests
// =============================================================================

func (s *TypesSuite) TestTypeName_Generic() {
	name := TypeName[*testTypesStruct]()
	s.Contains(name, "*", "should include pointer marker")
	s.Contains(name, "testTypesStruct", "should include type name")
}

func (s *TypesSuite) TestTypeName_GenericBuiltin() {
	name := TypeName[string]()
	s.Equal("string", name, "generic TypeName with builtin should return 'string'")
}

func (s *TypesSuite) TestTypeName_GenericSlice() {
	name := TypeName[[]byte]()
	s.Equal("[]uint8", name, "byte is alias for uint8")
}

// =============================================================================
// Test Helper Types
// =============================================================================

type testTypesStruct struct {
	value string
}
