package check

import (
	"regexp"
	"sort"
	"testing"

	"github.com/exograd/go-daemon/djson"
	"github.com/stretchr/testify/assert"
)

type testObj1 struct {
	A *testObj2
	B *testObj2
}

func (obj *testObj1) Check(c *Checker) {
	c.CheckObject("a", obj.A)
	c.CheckOptionalObject("b", obj.B)
}

type testObj2 struct {
	C int
}

func (obj *testObj2) Check(c *Checker) {
	c.CheckIntMin("c", obj.C, 1)
}

type testEnum string

const (
	testEnumFoo testEnum = "foo"
	testEnumBar testEnum = "bar"
)

var testEnumValues = []testEnum{testEnumFoo, testEnumBar}

func TestCheckTest(t *testing.T) {
	assert := assert.New(t)

	var c *Checker

	// Integers
	c = NewChecker()
	assert.True(c.CheckIntMin("t", 42, 1))
	assert.True(c.CheckIntMax("t", 42, 100))
	assert.True(c.CheckIntMinMax("t", 42, 1, 100))
	assert.False(c.CheckIntMinMax("t", 42, 100, 120))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(djson.Pointer{"t"}, c.Errors[0].Pointer)
	}

	// Strings
	c = NewChecker()
	assert.True(c.CheckStringLengthMin("t", "foo", 1))
	assert.True(c.CheckStringLengthMax("t", "foo", 10))
	assert.True(c.CheckStringLengthMinMax("t", "foo", 1, 10))
	assert.False(c.CheckStringLengthMinMax("t", "foo", 5, 10))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(djson.Pointer{"t"}, c.Errors[0].Pointer)
	}

	c = NewChecker()
	assert.True(c.CheckStringValue("t", "x", []string{"x", "y", "z"}))
	assert.False(c.CheckStringValue("t", "w", []string{"x", "y", "z"}))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(djson.Pointer{"t"}, c.Errors[0].Pointer)
	}

	c = NewChecker()
	re := regexp.MustCompile("^x")
	assert.True(c.CheckStringMatch("t", "x1", re))
	assert.False(c.CheckStringMatch("t", "y1", re))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(djson.Pointer{"t"}, c.Errors[0].Pointer)
	}

	c = NewChecker()
	assert.True(c.CheckStringURI("t", "http://example.com"))
	assert.False(c.CheckStringURI("t", ""))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(djson.Pointer{"t"}, c.Errors[0].Pointer)
	}

	// String types
	c = NewChecker()
	assert.True(c.CheckStringValue("t", testEnumFoo, testEnumValues))
	assert.False(c.CheckStringValue("t", "unknown", testEnumValues))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(djson.Pointer{"t"}, c.Errors[0].Pointer)
	}

	// Slices
	c = NewChecker()
	assert.True(c.CheckArrayLengthMin("t", []int{1, 2, 3}, 1))
	assert.True(c.CheckArrayLengthMax("t", []int{1, 2, 3}, 10))
	assert.True(c.CheckArrayLengthMinMax("t", []int{1, 2, 3}, 1, 10))
	assert.False(c.CheckArrayLengthMinMax("t", []int{1, 2, 3}, 5, 10))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(djson.Pointer{"t"}, c.Errors[0].Pointer)
	}

	// Arrays
	c = NewChecker()
	assert.True(c.CheckArrayLengthMin("t", [3]int{1, 2, 3}, 1))
	assert.True(c.CheckArrayLengthMax("t", [3]int{1, 2, 3}, 10))
	assert.True(c.CheckArrayLengthMinMax("t", [3]int{1, 2, 3}, 1, 10))
	assert.False(c.CheckArrayLengthMinMax("t", [3]int{1, 2, 3}, 5, 10))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(djson.Pointer{"t"}, c.Errors[0].Pointer)
	}

	// Objects
	c = NewChecker()
	obj1 := &testObj1{
		A: &testObj2{C: 1},
		B: &testObj2{C: 2},
	}
	assert.True(c.CheckObject("t", obj1))

	c = NewChecker()
	obj2 := &testObj1{
		A: &testObj2{C: 1},
		B: &testObj2{},
	}
	assert.False(c.CheckObject("t", obj2))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(djson.Pointer{"t", "b", "c"}, c.Errors[0].Pointer)
	}

	c = NewChecker()
	obj3 := &testObj1{
		A: &testObj2{C: 1},
		B: nil,
	}
	assert.True(c.CheckObject("t", obj3))

	c = NewChecker()
	obj4 := &testObj1{
		A: nil,
		B: &testObj2{C: 1},
	}
	assert.False(c.CheckObject("t", obj4))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(djson.Pointer{"t", "a"}, c.Errors[0].Pointer)
	}

	// Object arrays
	c = NewChecker()
	objArray1 := []*testObj2{
		&testObj2{C: 1},
		&testObj2{C: 2},
		&testObj2{C: 3},
	}
	assert.True(c.CheckObjectArray("t", objArray1))

	c = NewChecker()
	objArray2 := []*testObj2{
		&testObj2{C: 1},
		&testObj2{C: 2},
		&testObj2{C: 0},
		&testObj2{C: 3},
		&testObj2{C: 0},
	}
	assert.False(c.CheckObjectArray("t", objArray2))
	if assert.Equal(2, len(c.Errors)) {
		assert.Equal(djson.Pointer{"t", "2", "c"}, c.Errors[0].Pointer)
		assert.Equal(djson.Pointer{"t", "4", "c"}, c.Errors[1].Pointer)
	}

	// Object maps
	c = NewChecker()
	objMap1 := map[string]*testObj2{
		"v1": &testObj2{C: 1},
		"v2": &testObj2{C: 2},
		"v3": &testObj2{C: 3},
	}
	assert.True(c.CheckObjectMap("t", objMap1))

	c = NewChecker()
	objMap2 := map[string]*testObj2{
		"v1": &testObj2{C: 1},
		"v2": &testObj2{C: 2},
		"v3": &testObj2{C: 0},
		"v4": &testObj2{C: 3},
		"v5": &testObj2{C: 0},
	}
	assert.False(c.CheckObjectMap("t", objMap2))
	if assert.Equal(2, len(c.Errors)) {
		sort.Slice(c.Errors, func(i, j int) bool {
			return c.Errors[i].Pointer.String() < c.Errors[j].Pointer.String()
		})

		assert.Equal(djson.Pointer{"t", "v3", "c"}, c.Errors[0].Pointer)
		assert.Equal(djson.Pointer{"t", "v5", "c"}, c.Errors[1].Pointer)
	}
}
