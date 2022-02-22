// Copyright (c) 2022 Exograd SAS.
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY
// SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR
// IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package dcrypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPadPKCS5(t *testing.T) {
	assert := assert.New(t)

	assert.Equal([]byte("\x04\x04\x04\x04"),
		PadPKCS5([]byte(""), 4))
	assert.Equal([]byte("a\x03\x03\x03"),
		PadPKCS5([]byte("a"), 4))
	assert.Equal([]byte("ab\x02\x02"),
		PadPKCS5([]byte("ab"), 4))
	assert.Equal([]byte("abc\x01"),
		PadPKCS5([]byte("abc"), 4))
	assert.Equal([]byte("abcd\x04\x04\x04\x04"),
		PadPKCS5([]byte("abcd"), 4))
	assert.Equal([]byte("abcde\x03\x03\x03"),
		PadPKCS5([]byte("abcde"), 4))
	assert.Equal([]byte("abcdefgh\x04\x04\x04\x04"),
		PadPKCS5([]byte("abcdefgh"), 4))
}

func TestUnpadPKCS5(t *testing.T) {
	assert := assert.New(t)

	assertEqual := func(expected, data []byte) {
		t.Helper()

		data2, err := UnpadPKCS5(data, 4)
		if assert.NoError(err) {
			assert.Equal(expected, data2)
		}
	}

	assertEqual([]byte(""),
		[]byte("\x04\x04\x04\x04"))
	assertEqual([]byte("a"),
		[]byte("a\x03\x03\x03"))
	assertEqual([]byte("ab"),
		[]byte("ab\x02\x02"))
	assertEqual([]byte("abc"),
		[]byte("abc\x01"))
	assertEqual([]byte("abcd"),
		[]byte("abcd\x04\x04\x04\x04"))
	assertEqual([]byte("abcde"),
		[]byte("abcde\x03\x03\x03"))
	assertEqual([]byte("abcdefgh"),
		[]byte("abcdefgh\x04\x04\x04\x04"))
}
