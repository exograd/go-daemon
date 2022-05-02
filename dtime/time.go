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

package dtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const TimestampLayout = time.RFC3339

type Timestamp time.Time

func Now() Timestamp {
	return Timestamp(time.Now().UTC())
}

func (t Timestamp) String() string {
	return time.Time(t).Format(TimestampLayout)
}

func (t Timestamp) GoString() string {
	return time.Time(t).Format(TimestampLayout)
}

func (t *Timestamp) Parse(s string) error {
	tt, err := time.Parse(TimestampLayout, s)
	if err != nil {
		return errors.New("invalid format")
	}

	*t = Timestamp(tt.UTC())

	return nil
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	return t.Parse(s)
}

// database/sql.Scanner
func (t *Timestamp) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		return t.Parse(v)

	case []byte:
		return t.Parse(string(v))

	case time.Time:
		*t = Timestamp(v.UTC())
		return nil

	default:
		return fmt.Errorf("invalid timestamp value type %T", v)
	}
}
