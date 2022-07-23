// Copyright (c) 2022 Exograd SAS.
//
// Permission to use, copy, modify, and/or distribute this software for any
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

package dlog

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

type TerminalBackendCfg struct {
	Color       bool `json:"color"`
	DomainWidth int  `json:"domain_width"`
}

type TerminalBackend struct {
	Cfg TerminalBackendCfg

	domainWidth int
}

func NewTerminalBackend(cfg TerminalBackendCfg) *TerminalBackend {
	domainWidth := 24
	if cfg.DomainWidth > 0 {
		domainWidth = cfg.DomainWidth
	}

	isCharDev, err := IsCharDevice(os.Stderr)
	if err != nil {
		// If we cannot check for some reason, assume it is a character device
		isCharDev = true
	}

	if !isCharDev {
		cfg.Color = false
	}

	b := &TerminalBackend{
		Cfg: cfg,

		domainWidth: domainWidth,
	}

	return b
}

func (b *TerminalBackend) Log(msg Message) {
	domain := fmt.Sprintf("%-*s", b.domainWidth, msg.domain)

	level := string(msg.Level)
	if msg.Level == LevelDebug {
		level += "." + strconv.Itoa(msg.DebugLevel)
	}

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "%-7s  %s  %s\n",
		level, b.Colorize(ColorGreen, domain), msg.Message)

	if len(msg.Data) > 0 {
		fmt.Fprintf(&buf, "         ")

		keys := make([]string, len(msg.Data))
		i := 0
		for k := range msg.Data {
			keys[i] = k
			i++
		}
		sort.Strings(keys)

		for i, k := range keys {
			if i > 0 {
				fmt.Fprintf(&buf, " ")
			}

			fmt.Fprintf(&buf, "%s=%s",
				b.Colorize(ColorBlue, k), formatDatum(msg.Data[k]))

			i++
		}

		fmt.Fprintf(&buf, "\n")
	}

	io.Copy(os.Stderr, &buf)
}

func (b *TerminalBackend) Colorize(color Color, s string) string {
	if !b.Cfg.Color {
		return s
	}

	return Colorize(color, s)
}

func formatDatum(datum Datum) string {
	switch v := datum.(type) {
	case fmt.Stringer:
		return formatDatum(v.String())

	case string:
		if !strings.Contains(v, " ") {
			return v
		}

		return fmt.Sprintf("%q", v)

	default:
		return fmt.Sprintf("%v", v)
	}
}
