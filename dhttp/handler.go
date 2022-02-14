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
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/exograd/go-log"
)

type Handler struct {
	Server *Server

	Pattern string
	Method  string

	Request        *http.Request
	ResponseWriter http.ResponseWriter

	StartTime time.Time
}

func (h *Handler) Reply(status int, r io.Reader) {
	h.ResponseWriter.WriteHeader(status)

	if r != nil {
		if _, err := io.Copy(h.ResponseWriter, r); err != nil {
			h.Server.Log.Error("cannot write response: %v", err)
			return
		}
	}
}

func (h *Handler) logRequest() {
	req := h.Request
	w := h.ResponseWriter.(*ResponseWriter)

	reqTime := time.Since(h.StartTime)
	seconds := reqTime.Seconds()

	var reqTimeString string
	if seconds < 0.001 {
		reqTimeString = fmt.Sprintf("%dμs", int(math.Ceil(seconds*1e6)))
	} else if seconds < 1.0 {
		reqTimeString = fmt.Sprintf("%dms", int(math.Ceil(seconds*1e3)))
	} else {
		reqTimeString = fmt.Sprintf("%.1fs", seconds)
	}

	var resSizeString string
	if w.ResponseBodySize < 1000 {
		resSizeString = fmt.Sprintf("%dB", w.ResponseBodySize)
	} else if w.ResponseBodySize < 1_000_000 {
		resSizeString = fmt.Sprintf("%.1fKB", float64(w.ResponseBodySize)/1e3)
	} else if w.ResponseBodySize < 1_000_000_000 {
		resSizeString = fmt.Sprintf("%.1fMB", float64(w.ResponseBodySize)/1e6)
	} else {
		resSizeString = fmt.Sprintf("%.1fGB", float64(w.ResponseBodySize)/1e9)
	}

	data := log.Data{
		"method":        req.Method,
		"path":          req.URL.Path,
		"time":          reqTime.Microseconds(),
		"response_size": w.ResponseBodySize,
	}

	statusString := "-"
	if w.Status != 0 {
		statusString = strconv.Itoa(w.Status)
		data["status"] = w.Status
	}

	h.Server.Log.InfoData(data, "%s %s %s %s %s",
		req.Method, req.URL.Path, statusString, resSizeString, reqTimeString)
}
