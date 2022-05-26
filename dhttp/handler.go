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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"time"

	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-log"
	"github.com/go-chi/chi/v5"
)

type APIError struct {
	Message string       `json:"error"`
	Code    string       `json:"code,omitempty"`
	Data    APIErrorData `json:"data,omitempty"`
}

type APIErrorData map[string]interface{}

type Handler struct {
	Server *Server
	Log    *log.Logger

	ClientAddress string
	RequestId     string

	Pattern string
	Method  string
	RouteId string
	Query   url.Values

	Request        *http.Request
	ResponseWriter http.ResponseWriter

	StartTime time.Time

	errorCode string
}

func (h *Handler) RouteVariable(name string) string {
	return chi.URLParam(h.Request, name)
}

func (h *Handler) HasQueryParameter(name string) bool {
	return h.Query.Has(name)
}

func (h *Handler) QueryParameter(name string) string {
	return h.Query.Get(name)
}

func (h *Handler) RequestData() ([]byte, error) {
	data, err := ioutil.ReadAll(h.Request.Body)
	if err != nil {
		h.ReplyInternalError(500, "cannot read request body: %v", err)
		return nil, fmt.Errorf("cannot read request body: %w", err)
	}

	return data, nil
}

func (h *Handler) JSONRequestData(dest interface{}) error {
	data, err := h.RequestData()
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		h.ReplyError(400, "invalid_request_body",
			"invalid request body: %v", err)
		return fmt.Errorf("invalid request body: %w", err)
	}

	return nil
}

func (h *Handler) JSONRequestObject(obj check.Object) error {
	if err := h.JSONRequestData(obj); err != nil {
		return err
	}

	checker := check.NewChecker()

	obj.Check(checker)
	if err := checker.Error(); err != nil {
		h.ReplyRequestBodyValidationErrors(checker.Errors)
		return fmt.Errorf("invalid request body: %w", err)
	}

	return nil
}

func (h *Handler) ReplyRequestBodyValidationErrors(err check.ValidationErrors) {
	data := map[string]interface{}{
		"validation_errors": err,
	}

	h.ReplyErrorData(400, "invalid_request_body", data,
		"invalid request body:\n%v", err)
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

func (h *Handler) ReplyEmpty(status int) {
	h.Reply(status, nil)
}

func (h *Handler) ReplyRedirect(status int, uri string) {
	header := h.ResponseWriter.Header()
	header.Set("Location", uri)

	h.Reply(status, nil)
}

func (h *Handler) ReplyJSON(status int, value interface{}) {
	header := h.ResponseWriter.Header()
	header.Set("Content-Type", "application/json")

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(value); err != nil {
		h.Log.Error("cannot encode json response: %v", err)
		h.ResponseWriter.WriteHeader(500)
		return
	}

	h.Reply(status, &buf)
}

func (h *Handler) ReplyInternalError(status int, format string, args ...interface{}) {
	h.Log.Error("internal error: "+format, args...)
	h.ReplyError(status, "internal_error", "internal error")
}

func (h *Handler) ReplyNotImplemented(feature string) {
	h.ReplyError(501, "not_implemented", "%s not implemented", feature)
}

func (h *Handler) ReplyError(status int, code, format string, args ...interface{}) {
	h.errorCode = code
	h.Server.handleError(h, status, code, fmt.Sprintf(format, args...), nil)
}

func (h *Handler) ReplyErrorData(status int, code string, data APIErrorData, format string, args ...interface{}) {
	h.Server.handleError(h, status, code, fmt.Sprintf(format, args...), data)
}

func (h *Handler) handlePanic(value interface{}) string {
	var msg string

	switch v := value.(type) {
	case error:
		msg = v.Error()
	case string:
		msg = v
	default:
		msg = fmt.Sprintf("%#v", v)
	}

	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	buf = buf[0 : n-1]

	h.Log.Error("panic: %s\n%s", msg, string(buf))

	return msg
}

func (h *Handler) logRequest() {
	req := h.Request
	w := h.ResponseWriter.(*ResponseWriter)

	reqTime := time.Since(h.StartTime)
	seconds := reqTime.Seconds()

	var reqTimeString string
	if seconds < 0.001 {
		reqTimeString = fmt.Sprintf("%dµs", int(math.Ceil(seconds*1e6)))
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
		"time":          reqTime.Microseconds(),
		"response_size": w.ResponseBodySize,
	}

	statusString := "-"
	if w.Status != 0 {
		statusString = strconv.Itoa(w.Status)
		data["status"] = w.Status
	}

	if h.errorCode != "" {
		data["error"] = h.errorCode
	}

	h.Log.InfoData(data, "%s %s %s %s %s",
		req.Method, req.URL.Path, statusString, resSizeString, reqTimeString)
}
