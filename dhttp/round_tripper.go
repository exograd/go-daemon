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

package dhttp

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/exograd/go-log"
)

type RoundTripper struct {
	Cfg *ClientCfg
	Log *log.Logger

	http.RoundTripper
}

func NewRoundTripper(rt http.RoundTripper, cfg *ClientCfg) *RoundTripper {
	return &RoundTripper{
		Cfg: cfg,
		Log: cfg.Log,

		RoundTripper: rt,
	}
}

func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	rt.finalizeReq(req)

	res, err := rt.RoundTripper.RoundTrip(req)

	if err == nil && rt.Cfg.LogRequests {
		rt.logRequest(req, res, time.Since(start).Seconds())
	}

	return res, err
}

func (rt *RoundTripper) finalizeReq(req *http.Request) {
	for name, values := range rt.Cfg.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
}

func (rt *RoundTripper) logRequest(req *http.Request, res *http.Response, seconds float64) {
	var statusString string
	if res == nil {
		statusString = "-"
	} else {
		statusString = strconv.Itoa(res.StatusCode)
	}

	var reqTimeString string
	if seconds < 0.001 {
		reqTimeString = fmt.Sprintf("%dµs", int(math.Ceil(seconds*1e6)))
	} else if seconds < 1.0 {
		reqTimeString = fmt.Sprintf("%dms", int(math.Ceil(seconds*1e3)))
	} else {
		reqTimeString = fmt.Sprintf("%.1fs", seconds)
	}

	rt.Log.Info("%s %s %s %s", req.Method, req.URL.String(), statusString,
		reqTimeString)
}
