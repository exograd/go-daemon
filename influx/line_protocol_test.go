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

package influx

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEncodePoint(t *testing.T) {
	assert := assert.New(t)

	timestamp := time.Now().UTC()

	tests := []struct {
		p    *Point
		line string
	}{
		{NewPoint("m1", Tags{}, Fields{"a": 1}),
			`m1 a=1i`},
		{NewPoint("m2", Tags{}, Fields{"a": 123, "b": true, "c": "foo"}),
			`m2 a=123i,b=true,c="foo"`},
		{NewPoint("m3", Tags{"x": "foo"}, Fields{"a": -1}),
			`m3,x=foo a=-1i`},
		{NewPoint("m4", Tags{"x": "1", "y": "23"}, Fields{"abc": "def"}),
			`m4,x=1,y=23 abc="def"`},
		{NewPointWithTimestamp("m5", Tags{}, Fields{"a": 1}, &timestamp),
			`m5 a=1i ` + strconv.FormatInt(timestamp.UnixNano(), 10)},
		{NewPoint(" m, 6 ", Tags{", =": `""`}, Fields{"=": `"a"`}),
			`\ m\,\ 6\ ,\,\ \=="" \=="\"a\""`},
	}

	for _, test := range tests {
		var buf bytes.Buffer
		EncodePoint(test.p, &buf)
		assert.Equal(test.line, buf.String(), test.p.Measurement)
	}
}

func TestEncodePoints(t *testing.T) {
	assert := assert.New(t)

	timestamp := time.Now().UTC()

	tests := []struct {
		ps   Points
		line string
	}{
		{Points{},
			""},
		{Points{
			NewPoint("m1", Tags{}, Fields{"a": 1}),
		},
			"m1 a=1i\n"},
		{Points{
			NewPoint("m1", Tags{}, Fields{"a": 1}),
			NewPoint("m2", Tags{"x": "foo"}, Fields{"a": 1, "b": false}),
		},
			"m1 a=1i\nm2,x=foo a=1i,b=false\n"},
		{Points{
			NewPoint("m1", Tags{}, Fields{"a": 1}),
			NewPoint("m2", Tags{"x": "foo"}, Fields{"a": 1, "b": false}),
			NewPointWithTimestamp("m3", Tags{}, Fields{"a": "n"}, &timestamp),
		},
			"m1 a=1i\nm2,x=foo a=1i,b=false\nm3 a=\"n\" " +
				strconv.FormatInt(timestamp.UnixNano(), 10) + "\n"},
	}

	for i, test := range tests {
		var buf bytes.Buffer
		EncodePoints(test.ps, &buf)
		assert.Equal(test.line, buf.String(), i+1)
	}
}
