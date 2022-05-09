package jsonpointer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPointerParse(t *testing.T) {
	assert := assert.New(t)

	assertParse := func(ep Pointer, s string) {
		t.Helper()

		var p Pointer
		if assert.NoError(p.Parse(s), s) {
			assert.Equal(ep, p, s)
		}
	}

	assertParse(Pointer{}, "")
	assertParse(Pointer{"foo"}, "/foo")
	assertParse(Pointer{"foo", "bar"}, "/foo/bar")
	assertParse(Pointer{"a", "b", "c"}, "/a/b/c")
	assertParse(Pointer{"xy", "", "z", "", ""}, "/xy//z//")
	assertParse(Pointer{"foo/bar", "~hello"}, "/foo~1bar/~0hello")
	assertParse(Pointer{"~1", "/0"}, "/~01/~10")
}

func TestPointerString(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("", Pointer{}.String())
	assert.Equal("/foo", Pointer{"foo"}.String())
	assert.Equal("/foo/bar", Pointer{"foo", "bar"}.String())
	assert.Equal("/a/b/c", Pointer{"a", "b", "c"}.String())
	assert.Equal("/xy//z//", Pointer{"xy", "", "z", "", ""}.String())
	assert.Equal("/foo~1bar/~0hello", Pointer{"foo/bar", "~hello"}.String())
	assert.Equal("/~01/~10", Pointer{"~1", "/0"}.String())
}
