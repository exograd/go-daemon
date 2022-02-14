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
	"math"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/exograd/go-log"
	"github.com/go-chi/chi/v5"
)

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
	w2 := NewResponseWriter(w)

	startTime := time.Now()
	defer s.logRequest(req, w2, startTime)

	s.Router.ServeHTTP(w2, req)
}

func (s *Server) logRequest(req *http.Request, w *ResponseWriter, startTime time.Time) {
	reqTime := time.Since(startTime)
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
		"status":        w.Status,
		"time":          reqTime.Microseconds(),
		"response_size": w.ResponseBodySize,
	}

	s.Log.InfoData(data, "%s %s %d %s %s",
		req.Method, req.URL.Path, w.Status, resSizeString, reqTimeString)
}
