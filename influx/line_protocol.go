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
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	measurementReplacer *strings.Replacer
	keyReplacer         *strings.Replacer
	stringFieldReplacer *strings.Replacer
)

func init() {
	measurementReplacer = strings.NewReplacer(`,`, `\,`, ` `, `\ `)
	keyReplacer = strings.NewReplacer(`,`, `\,`, `=`, `\=`, ` `, `\ `)
	stringFieldReplacer = strings.NewReplacer(`"`, `\"`)
}

func EncodePoint(p *Point, buf *bytes.Buffer) {
	encodeMeasurement(p.Measurement, buf)
	if len(p.Tags) > 0 {
		encodeTags(p.Tags, buf)
	}

	buf.WriteByte(' ')
	encodeFields(p.Fields, buf)

	if p.Timestamp != nil {
		buf.WriteByte(' ')
		encodeTimestamp(p.Timestamp, buf)
	}
}

func EncodePoints(ps Points, buf *bytes.Buffer) {
	for _, p := range ps {
		EncodePoint(p, buf)
		buf.WriteByte('\n')
	}
}

func encodeMeasurement(measurement string, buf *bytes.Buffer) {
	measurementReplacer.WriteString(buf, measurement)
}

func encodeTags(tags Tags, buf *bytes.Buffer) {
	// From the InfluxDB documentation:
	//
	// For best performance you should sort tags by key before sending them to
	// the database. The sort should match the results from the Go
	// bytes.Compare function.

	keys := make([]string, len(tags))
	i := 0
	for key := range tags {
		keys[i] = key
		i++
	}

	sort.Strings(keys)

	for _, key := range keys {
		buf.WriteByte(',')
		encodeKey(key, buf)
		buf.WriteByte('=')
		encodeKey(tags[key], buf)
	}
}

func encodeFields(fields Fields, buf *bytes.Buffer) {
	// While not required, we sort fields to make life easier for tests.

	keys := make([]string, len(fields))
	i := 0
	for key := range fields {
		keys[i] = key
		i++
	}

	sort.Strings(keys)

	for i, key := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}

		encodeKey(key, buf)
		buf.WriteByte('=')
		encodeFieldValue(fields[key], buf)
	}
}

func encodeKey(key string, buf *bytes.Buffer) {
	keyReplacer.WriteString(buf, key)
}

func encodeFieldValue(value interface{}, buf *bytes.Buffer) {
	switch v := value.(type) {
	case float32:
		buf.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 32))
	case float64:
		buf.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
	case int, int8, int16, int32, int64:
		fmt.Fprintf(buf, "%di", v)
	case uint, uint8, uint16, uint32, uint64:
		fmt.Fprintf(buf, "%di", v)
	case bool:
		fmt.Fprintf(buf, "%v", v)
	case string:
		buf.WriteByte('"')
		fmt.Printf("XXX %s → %s\n", v, stringFieldReplacer.Replace(v))
		stringFieldReplacer.WriteString(buf, v)
		buf.WriteByte('"')
	case []byte:
		encodeFieldValue(string(v), buf)
	default:
		encodeFieldValue(fmt.Sprintf("%v", v), buf)
	}
}

func encodeTimestamp(timestamp *time.Time, buf *bytes.Buffer) {
	ns := timestamp.UnixNano()
	fmt.Fprintf(buf, "%d", ns)
}
