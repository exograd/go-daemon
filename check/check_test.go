package check

import (
	"regexp"
	"testing"

	"github.com/exograd/go-daemon/jsonpointer"
	"github.com/stretchr/testify/assert"
)

type checkTestObj1 struct {
	A *checkTestObj2
	B *checkTestObj2
}

func (obj *checkTestObj1) Check(c *Checker) {
	c.CheckObject("a", obj.A)
	c.CheckOptionalObject("b", obj.B)
}

type checkTestObj2 struct {
	C int
}

func (obj *checkTestObj2) Check(c *Checker) {
	c.CheckIntMin("c", obj.C, 1)
}

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
		assert.Equal(jsonpointer.Pointer{"t"}, c.Errors[0].Pointer)
	}

	// Strings
	c = NewChecker()
	assert.True(c.CheckStringLengthMin("t", "foo", 1))
	assert.True(c.CheckStringLengthMax("t", "foo", 10))
	assert.True(c.CheckStringLengthMinMax("t", "foo", 1, 10))
	assert.False(c.CheckStringLengthMinMax("t", "foo", 5, 10))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(jsonpointer.Pointer{"t"}, c.Errors[0].Pointer)
	}

	c = NewChecker()
	assert.True(c.CheckStringValue("t", "x", []string{"x", "y", "z"}))
	assert.False(c.CheckStringValue("t", "w", []string{"x", "y", "z"}))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(jsonpointer.Pointer{"t"}, c.Errors[0].Pointer)
	}

	c = NewChecker()
	re := regexp.MustCompile("^x")
	assert.True(c.CheckStringMatch("t", "x1", re))
	assert.False(c.CheckStringMatch("t", "y1", re))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(jsonpointer.Pointer{"t"}, c.Errors[0].Pointer)
	}

	// Slices
	c = NewChecker()
	assert.True(c.CheckArrayLengthMin("t", []int{1, 2, 3}, 1))
	assert.True(c.CheckArrayLengthMax("t", []int{1, 2, 3}, 10))
	assert.True(c.CheckArrayLengthMinMax("t", []int{1, 2, 3}, 1, 10))
	assert.False(c.CheckArrayLengthMinMax("t", []int{1, 2, 3}, 5, 10))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(jsonpointer.Pointer{"t"}, c.Errors[0].Pointer)
	}

	// Arrays
	c = NewChecker()
	assert.True(c.CheckArrayLengthMin("t", [3]int{1, 2, 3}, 1))
	assert.True(c.CheckArrayLengthMax("t", [3]int{1, 2, 3}, 10))
	assert.True(c.CheckArrayLengthMinMax("t", [3]int{1, 2, 3}, 1, 10))
	assert.False(c.CheckArrayLengthMinMax("t", [3]int{1, 2, 3}, 5, 10))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(jsonpointer.Pointer{"t"}, c.Errors[0].Pointer)
	}

	// Objects
	c = NewChecker()
	obj1 := &checkTestObj1{
		A: &checkTestObj2{C: 1},
		B: &checkTestObj2{C: 2},
	}
	assert.True(c.CheckObject("t", obj1))

	c = NewChecker()
	obj2 := &checkTestObj1{
		A: &checkTestObj2{C: 1},
		B: &checkTestObj2{},
	}
	assert.False(c.CheckObject("t", obj2))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(jsonpointer.Pointer{"t", "b", "c"}, c.Errors[0].Pointer)
	}

	c = NewChecker()
	obj3 := &checkTestObj1{
		A: &checkTestObj2{C: 1},
		B: nil,
	}
	assert.True(c.CheckObject("t", obj3))

	c = NewChecker()
	obj4 := &checkTestObj1{
		A: nil,
		B: &checkTestObj2{C: 1},
	}
	assert.False(c.CheckObject("t", obj4))
	if assert.Equal(1, len(c.Errors)) {
		assert.Equal(jsonpointer.Pointer{"t", "a"}, c.Errors[0].Pointer)
	}
}
