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
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/dlog"
	"github.com/exograd/go-daemon/ksuid"
	"github.com/go-chi/chi/v5"
)

type contextKey struct{}

var (
	contextKeyHandler contextKey = struct{}{}
)

type RouteFunc func(*Handler)

type ErrorHandler func(*Handler, int, string, string, APIErrorData)

type ServerCfg struct {
	Log *dlog.Logger `json:"-"`

	ErrorHandler ErrorHandler `json:"-"`

	Address string `json:"address"`

	TLS *TLSServerCfg `json:"tls,omitempty"`

	HideInternalErrors bool `json:"hide_internal_errors"`
}

type TLSServerCfg struct {
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
}

type Server struct {
	Cfg ServerCfg
	Log *dlog.Logger

	server *http.Server
	Router *chi.Mux

	stopChan  chan struct{}
	errorChan chan error
	wg        sync.WaitGroup
}

func (cfg *ServerCfg) Check(c *check.Checker) {
	c.CheckStringNotEmpty("address", cfg.Address)
	c.CheckOptionalObject("tls", cfg.TLS)
}

func (cfg *TLSServerCfg) Check(c *check.Checker) {
	c.CheckStringNotEmpty("certificate", cfg.Certificate)
	c.CheckStringNotEmpty("private_key", cfg.PrivateKey)
}

func NewServer(cfg ServerCfg) (*Server, error) {
	if cfg.Log == nil {
		cfg.Log = dlog.DefaultLogger("http-server")
	}

	if cfg.Address == "" {
		cfg.Address = "localhost:8080"
	}

	s := &Server{
		Cfg: cfg,
		Log: cfg.Log,

		stopChan:  make(chan struct{}),
		errorChan: make(chan error),
	}

	s.Router = chi.NewMux()
	s.Router.NotFound(s.handleNotFound)
	s.Router.MethodNotAllowed(s.handleMethodNotAllowed)

	s.server = &http.Server{
		Addr:     cfg.Address,
		Handler:  s,
		ErrorLog: s.Log.StdLogger(dlog.LevelError),
	}

	if cfg.TLS != nil {
		s.server.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS13,
			PreferServerCipherSuites: true,
		}
	}

	return s, nil
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.Cfg.Address)
	if err != nil {
		return fmt.Errorf("cannot listen on %q: %w", s.Cfg.Address, err)
	}

	s.Log.Info("listening on %q", s.Cfg.Address)

	go func() {
		var err error

		if s.Cfg.TLS == nil {
			err = s.server.Serve(listener)
		} else {
			certificate := s.Cfg.TLS.Certificate
			privateKey := s.Cfg.TLS.PrivateKey

			err = s.server.ServeTLS(listener, certificate, privateKey)
		}

		if err != nil {
			if err != http.ErrServerClosed {
				s.errorChan <- err
			}
		}
	}()

	s.wg.Add(1)
	go s.main()

	return nil
}

func (s *Server) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

func (s *Server) Terminate() {
	close(s.errorChan)
}

func (s *Server) main() {
	defer func() {
		s.wg.Done()
	}()

	select {
	case <-s.stopChan:
		s.shutdown()

	case err := <-s.errorChan:
		s.Log.Error("server error: %v", err)
	}
}

func (s *Server) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		s.Log.Error("cannot shutdown server: %v", err)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h := &Handler{
		Server: s,
		Log:    s.Log.Child("", nil),

		StartTime: time.Now(),
	}

	ctx := req.Context()
	ctx = context.WithValue(ctx, contextKeyHandler, h)

	h.Request = req.WithContext(ctx)
	h.ResponseWriter = NewResponseWriter(w)

	h.ClientAddress = requestClientAddress(req)
	h.Log.Data["address"] = h.ClientAddress

	h.RequestId = requestId(req)
	if h.RequestId == "" {
		h.RequestId = ksuid.Generate().String()
	}
	h.Log.Data["request_id"] = h.RequestId

	h.Query = req.URL.Query()

	defer h.logRequest()

	defer func() {
		if value := recover(); value != nil {
			msg := h.handlePanic(value)
			h.ReplyInternalError(500, "panic: %s", msg)
		}
	}()

	s.Router.ServeHTTP(h.ResponseWriter, h.Request)
}

func (s *Server) Route(pattern, method string, routeFunc RouteFunc) {
	handlerFunc := func(w http.ResponseWriter, req *http.Request) {
		h := requestHandler(req)
		h.Request = req // the request object was modified by chi

		routeId := pattern + " " + method
		h.Log.Data["route_id"] = routeId

		h.Pattern = pattern
		h.Method = method
		h.RouteId = routeId

		routeFunc(h)
	}

	s.Router.MethodFunc(method, pattern, handlerFunc)
}

func (s *Server) handleError(h *Handler, status int, code, msg string, data APIErrorData) {
	if s.Cfg.ErrorHandler == nil {
		h.ReplyJSON(status, APIError{Message: msg, Code: code, Data: data})
		return
	}

	s.Cfg.ErrorHandler(h, status, code, msg, data)
}

func (s *Server) handleNotFound(w http.ResponseWriter, req *http.Request) {
	h := requestHandler(req)

	h.ReplyError(404, "route_not_found", "route not found")
}

func (s *Server) handleMethodNotAllowed(w http.ResponseWriter, req *http.Request) {
	h := requestHandler(req)

	h.ReplyError(405, "unhandled_method", "unhandled method")
}

func requestClientAddress(req *http.Request) string {
	if v := req.Header.Get("X-Real-IP"); v != "" {
		return v
	} else if v := req.Header.Get("X-Forwarded-For"); v != "" {
		i := strings.Index(v, ", ")
		if i == -1 {
			return v
		}

		return v[:i]
	} else {
		host, _, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			return ""
		}

		return host
	}
}

func requestId(req *http.Request) string {
	return req.Header.Get("X-Request-Id")
}

func requestHandler(req *http.Request) *Handler {
	value := req.Context().Value(contextKeyHandler)
	if value == nil {
		return nil
	}

	return value.(*Handler)
}
