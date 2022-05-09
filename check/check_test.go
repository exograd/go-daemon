package check

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCheckObject struct {
	A int
	B string

	C1 *testCheckSubObject1
	C2 *testCheckSubObject1

	D1 *testCheckSubObject2
	D2 *testCheckSubObject2
}

type testCheckSubObject1 struct {
	A1 int
}

type testCheckSubObject2 struct {
	A2 int
}

func (obj *testCheckSubObject2) Check(c *Checker) {
	c.CheckIntMax("a2", obj.A2, 100)
}

func TestCheckTest(t *testing.T) {
	assert := assert.New(t)

	var c *Checker

	obj := testCheckObject{
		A: 42,
		B: "foo",
		C1: &testCheckSubObject1{
			A1: 123,
		},
		D1: &testCheckSubObject2{
			A2: 456,
		},
	}

	c = NewChecker()

	assert.True(c.CheckIntMinMax("a", obj.A, 1, 100))
	assert.True(c.CheckStringLengthMinMax("b", obj.B, 3, 5))
	assert.True(c.CheckObject("c1", obj.C1))
	assert.True(c.CheckOptionalObject("c1", obj.C1))
	assert.True(c.CheckOptionalObject("c2", obj.C2))

	assert.False(c.CheckIntMax("a", obj.A, 10))
	assert.False(c.CheckStringLengthMin("b", obj.B, 5))
	assert.False(c.CheckObject("c2", obj.C2))
	assert.False(c.CheckObject("d1", obj.D1))

	assert.Equal(4, len(c.Errors))
	assert.Equal([]string{"a"}, c.Errors[0].Path)
	assert.Equal([]string{"b"}, c.Errors[1].Path)
	assert.Equal([]string{"c2"}, c.Errors[2].Path)
	assert.Equal([]string{"d1", "a2"}, c.Errors[3].Path)
}
