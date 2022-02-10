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
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/exograd/go-daemon/dhttp"
	"github.com/exograd/go-daemon/influx"
	"github.com/exograd/go-log"
	"github.com/exograd/go-program"
)

type DaemonCfg struct {
	name        string
	description string

	Logger *log.LoggerCfg

	HTTPServers map[string]dhttp.ServerCfg
	HTTPClients map[string]dhttp.ClientCfg

	Influx *influx.ClientCfg
}

func NewDaemonCfg() DaemonCfg {
	return DaemonCfg{
		HTTPServers: make(map[string]dhttp.ServerCfg),
		HTTPClients: make(map[string]dhttp.ClientCfg),
	}
}

type Daemon struct {
	Cfg DaemonCfg

	Log *log.Logger

	program *program.Program
	service Service

	httpServers map[string]*dhttp.Server
	httpClients map[string]*dhttp.Client

	Influx *influx.Client

	Hostname string

	stopChan  chan struct{}
	errorChan chan error
	wg        sync.WaitGroup
}

func newDaemon(cfg DaemonCfg, p *program.Program, service Service) *Daemon {
	d := &Daemon{
		Cfg: cfg,

		program: p,
		service: service,

		stopChan:  make(chan struct{}, 1),
		errorChan: make(chan error, 1),
	}

	return d
}

func (d *Daemon) init() error {
	d.initDefaultLogger()

	initFuncs := []func() error{
		d.initHostname,
		d.initLogger,
		d.initHTTPServers,
		d.initHTTPClients,
		d.initInflux,
	}

	for _, initFunc := range initFuncs {
		if err := initFunc(); err != nil {
			return err
		}
	}

	if err := d.service.Init(d); err != nil {
		return err
	}

	return nil
}

func (d *Daemon) initDefaultLogger() {
	d.Log = log.DefaultLogger(d.Cfg.name)
}

func (d *Daemon) initHostname() error {
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("cannot obtain hostname: %w", err)
	}

	d.Hostname = hostname

	return nil
}

func (d *Daemon) initLogger() error {
	if d.Cfg.Logger == nil {
		return nil
	}

	logger, err := log.NewLogger(d.Cfg.name, *d.Cfg.Logger)
	if err != nil {
		return fmt.Errorf("invalid logger configuration: %w", err)
	}

	d.Log = logger

	return nil
}

func (d *Daemon) initHTTPServers() error {
	d.httpServers = make(map[string]*dhttp.Server)

	for name, cfg := range d.Cfg.HTTPServers {
		cfg.Log = d.Log.Child("http-server", log.Data{"server": name})

		server, err := dhttp.NewServer(cfg)
		if err != nil {
			return fmt.Errorf("cannot create http server %q: %w", name, err)
		}

		d.httpServers[name] = server
	}

	return nil
}

func (d *Daemon) initHTTPClients() error {
	d.httpClients = make(map[string]*dhttp.Client)

	if d.Cfg.Influx != nil {
		cfg := influx.HTTPClientCfg(d.Cfg.Influx)

		if err := d.initHTTPClient("influx", cfg); err != nil {
			return err
		}
	}

	for name, cfg := range d.Cfg.HTTPClients {
		if err := d.initHTTPClient(name, cfg); err != nil {
			return err
		}
	}

	return nil
}

func (d *Daemon) initHTTPClient(name string, cfg dhttp.ClientCfg) error {
	cfg.Log = d.Log.Child("http-client", log.Data{"client": name})

	client, err := dhttp.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("cannot create http client %q: %w", name, err)
	}

	if _, found := d.httpClients[name]; found {
		return fmt.Errorf("duplicate http client %q", name)
	}

	d.httpClients[name] = client

	return nil
}

func (d *Daemon) initInflux() error {
	if d.Cfg.Influx == nil {
		return nil
	}

	cfg := *d.Cfg.Influx

	cfg.Log = d.Log.Child("influx", log.Data{})
	cfg.HTTPClient = d.httpClients["influx"]
	cfg.Hostname = d.Hostname

	client, err := influx.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("cannot create influx client: %w", err)
	}

	d.Influx = client

	return nil
}

func (d *Daemon) wait() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case signo := <-sigChan:
		fmt.Println()
		d.Log.Info("received signal %d (%v)", signo, signo)

	case <-d.stopChan:

	case err := <-d.errorChan:
		d.Log.Error("daemon error: %v", err)
		d.program.Fatal("daemon error: %v", err)
	}
}

func (d *Daemon) start() error {
	d.Log.Info("starting")

	if err := d.service.Start(d); err != nil {
		return err
	}

	for name, s := range d.httpServers {
		if err := s.Start(); err != nil {
			return fmt.Errorf("cannot start http server %q: %w", name, err)
		}
	}

	if d.Influx != nil {
		d.Influx.Start()
	}

	d.Log.Info("started")

	return nil
}

func (d *Daemon) stop() {
	d.Log.Info("stopping")

	if d.Influx != nil {
		d.Influx.Stop()
	}

	for _, s := range d.httpServers {
		s.Stop()
	}

	d.service.Stop(d)

	d.Log.Info("stopped")
}

func (d *Daemon) terminate() {
	if d.Influx != nil {
		d.Influx.Terminate()
	}

	for _, c := range d.httpClients {
		c.Terminate()
	}

	for _, s := range d.httpServers {
		s.Terminate()
	}

	d.service.Terminate(d)

	close(d.stopChan)
	close(d.errorChan)
}

func (d *Daemon) fatal(err error) {
	d.errorChan <- err
}

func Run(name, description string, service Service) {
	// Program
	p := program.NewProgram(name, description)

	p.AddOption("c", "cfg-file", "path", "",
		"the path of the configuration file")

	p.ParseCommandLine()

	// Configuration
	serviceCfg := service.ServiceCfg()

	if p.IsOptionSet("cfg-file") {
		cfgPath := p.OptionValue("cfg-file")

		if err := LoadCfg(cfgPath, serviceCfg); err != nil {
			p.Fatal("cannot load configuration: %v", err)
		}
	}

	daemonCfg, err := service.DaemonCfg()
	if err != nil {
		p.Fatal("invalid configuration: %v", err)
	}

	daemonCfg.name = name
	daemonCfg.description = description

	// Daemon
	d := newDaemon(daemonCfg, p, service)

	if err := d.init(); err != nil {
		p.Fatal("cannot initialize daemon: %v", err)
	}

	if err := d.start(); err != nil {
		p.Fatal("cannot start daemon: %v", err)
	}

	d.wait()
	d.stop()

	d.terminate()
}
