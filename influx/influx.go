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

import "time"

type Point struct {
	Measurement string
	Tags        Tags
	Fields      Fields
	Timestamp   *time.Time
}

type Points []*Point

type Tags map[string]string

type Fields map[string]interface{}

func NewPoint(measurement string, tags Tags, fields Fields) *Point {
	return &Point{
		Measurement: measurement,
		Tags:        tags,
		Fields:      fields,
	}
}

func NewPointWithTimestamp(measurement string, tags Tags, fields Fields, t *time.Time) *Point {
	return &Point{
		Measurement: measurement,
		Tags:        tags,
		Fields:      fields,
		Timestamp:   t,
	}
}
