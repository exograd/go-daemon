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
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/exograd/go-log"
	"github.com/go-chi/chi/v5"
)

type contextKey struct{}

var (
	contextKeyHandler contextKey = struct{}{}
)

type RouteFunc func(*Handler)

type ServerCfg struct {
	Log *log.Logger `json:"-"`

	Address string `json:"address"`
}

type Server struct {
	Cfg ServerCfg
	Log *log.Logger

	server *http.Server
	Router *chi.Mux

	stopChan  chan struct{}
	errorChan chan error
	wg        sync.WaitGroup
}

func NewServer(cfg ServerCfg) (*Server, error) {
	if cfg.Log == nil {
		cfg.Log = log.DefaultLogger("http-server")
	}

	if cfg.Address == "" {
		cfg.Address = "localhost:8080"
	}

	s := &Server{
		Cfg: cfg,
		Log: cfg.Log,

		Router: chi.NewMux(),

		stopChan:  make(chan struct{}),
		errorChan: make(chan error),
	}

	s.server = &http.Server{
		Addr:    cfg.Address,
		Handler: s,
	}

	return s, nil
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.Cfg.Address)
	if err != nil {
		return fmt.Errorf("cannot listen on %q: %w", s.Cfg.Address, err)
	}

	s.Log.Info("listening on %s", s.Cfg.Address)

	go func() {
		if err := s.server.Serve(listener); err != nil {
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
	s.wg.Done()
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

	defer h.logRequest()

	s.Router.ServeHTTP(h.ResponseWriter, h.Request)
}

func (s *Server) Route(pattern, method string, routeFunc RouteFunc) {
	handlerFunc := func(w http.ResponseWriter, req *http.Request) {
		h := req.Context().Value(contextKeyHandler).(*Handler)

		routeId := pattern + " " + method
		h.Log.Data["route_id"] = routeId

		h.Pattern = pattern
		h.Method = method
		h.RouteId = routeId

		routeFunc(h)
	}

	s.Router.MethodFunc(method, pattern, handlerFunc)
}
