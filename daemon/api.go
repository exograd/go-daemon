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

package daemon

import (
	"github.com/exograd/go-daemon/check"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	DefaultAPIAddress = "localhost:4196"
)

type APICfg struct {
	Address string `json:"address"`
}

func (cfg *APICfg) Check(c *check.Checker) {
	// We do not check that the address is not empty since we accept an empty
	// value and replace it with DefaultAPIAddress.
}

func (d *Daemon) initAPI() error {
	if d.Cfg.API == nil {
		return nil
	}

	server := d.HTTPServers["daemon-api"]

	server.Router.Mount("/debug", middleware.Profiler())

	return nil
}
