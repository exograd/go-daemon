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
	"syscall"

	"github.com/exograd/go-daemon/dhttp"
	"github.com/exograd/go-daemon/dlog"
	"github.com/exograd/go-daemon/influx"
	"github.com/exograd/go-daemon/pg"
	"github.com/exograd/go-program"
)

type DaemonCfg struct {
	name string

	Logger *dlog.LoggerCfg

	API *APICfg

	HTTPServers map[string]dhttp.ServerCfg
	HTTPClients map[string]dhttp.ClientCfg

	Influx *influx.ClientCfg

	Pg *pg.ClientCfg
}

func NewDaemonCfg() DaemonCfg {
	return DaemonCfg{
		HTTPServers: make(map[string]dhttp.ServerCfg),
		HTTPClients: make(map[string]dhttp.ClientCfg),
	}
}

func (cfg DaemonCfg) AddHTTPServer(name string, serverCfg dhttp.ServerCfg) {
	if _, found := cfg.HTTPServers[name]; found {
		panic(fmt.Sprintf("duplicate http server %q", name))
	}

	cfg.HTTPServers[name] = serverCfg
}

func (cfg DaemonCfg) AddHTTPClient(name string, clientCfg dhttp.ClientCfg) {
	if _, found := cfg.HTTPClients[name]; found {
		panic(fmt.Sprintf("duplicate http client %q", name))
	}

	cfg.HTTPClients[name] = clientCfg
}

type Daemon struct {
	Cfg DaemonCfg

	Log *dlog.Logger

	service Service

	HTTPServers map[string]*dhttp.Server
	HTTPClients map[string]*dhttp.Client

	Influx *influx.Client

	Pg *pg.Client

	Hostname string

	stopChan  chan struct{}
	errorChan chan error
}

func newDaemon(cfg DaemonCfg, service Service) *Daemon {
	d := &Daemon{
		Cfg: cfg,

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
		d.initPg,
		d.initAPI,
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
	d.Log = dlog.DefaultLogger(d.Cfg.name)
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

	logger, err := dlog.NewLogger(d.Cfg.name, *d.Cfg.Logger)
	if err != nil {
		return fmt.Errorf("invalid logger configuration: %w", err)
	}

	d.Log = logger

	return nil
}

func (d *Daemon) initHTTPServers() error {
	if apiCfg := d.Cfg.API; apiCfg != nil {
		address := apiCfg.Address
		if address == "" {
			address = DefaultAPIAddress
		}

		d.Cfg.AddHTTPServer("daemon-api", dhttp.ServerCfg{
			Address: address,
		})
	}

	d.HTTPServers = make(map[string]*dhttp.Server)

	for name, cfg := range d.Cfg.HTTPServers {
		cfg.Log = d.Log.Child("http-server", dlog.Data{"server": name})

		server, err := dhttp.NewServer(cfg)
		if err != nil {
			return fmt.Errorf("cannot create http server %q: %w", name, err)
		}

		d.HTTPServers[name] = server
	}

	return nil
}

func (d *Daemon) initHTTPClients() error {
	d.HTTPClients = make(map[string]*dhttp.Client)

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
	cfg.Log = d.Log.Child("http-client", dlog.Data{"client": name})

	client, err := dhttp.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("cannot create http client %q: %w", name, err)
	}

	if _, found := d.HTTPClients[name]; found {
		return fmt.Errorf("duplicate http client %q", name)
	}

	d.HTTPClients[name] = client

	return nil
}

func (d *Daemon) initInflux() error {
	if d.Cfg.Influx == nil {
		return nil
	}

	cfg := *d.Cfg.Influx

	cfg.Log = d.Log.Child("influx", dlog.Data{})
	cfg.HTTPClient = d.HTTPClients["influx"]
	cfg.Hostname = d.Hostname

	client, err := influx.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("cannot create influx client: %w", err)
	}

	d.Influx = client

	return nil
}

func (d *Daemon) initPg() error {
	if d.Cfg.Pg == nil {
		return nil
	}

	cfg := *d.Cfg.Pg

	cfg.Log = d.Log.Child("pg", dlog.Data{})

	client, err := pg.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("cannot create pg client: %w", err)
	}

	d.Pg = client

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
		os.Exit(1)
	}
}

func (d *Daemon) start() error {
	d.Log.Info("starting")

	for name, s := range d.HTTPServers {
		if err := s.Start(); err != nil {
			return fmt.Errorf("cannot start http server %q: %w", name, err)
		}
	}

	if d.Influx != nil {
		d.Influx.Start()
	}

	if err := d.service.Start(d); err != nil {
		return err
	}

	d.Log.Info("started")

	return nil
}

func (d *Daemon) stop() {
	d.Log.Info("stopping")

	d.service.Stop(d)

	if d.Pg != nil {
		d.Pg.Close()
	}

	if d.Influx != nil {
		d.Influx.Stop()
	}

	for _, s := range d.HTTPServers {
		s.Stop()
	}

	d.Log.Info("stopped")
}

func (d *Daemon) terminate() {
	d.service.Terminate(d)

	if d.Influx != nil {
		d.Influx.Terminate()
	}

	for _, c := range d.HTTPClients {
		c.Terminate()
	}

	for _, s := range d.HTTPServers {
		s.Terminate()
	}

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

	// Daemon
	d := newDaemon(daemonCfg, service)

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

func RunTest(name string, service Service, cfgPath string, readyChan chan<- struct{}) {
	abort := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
		os.Exit(1)
	}

	// Configuration
	serviceCfg := service.ServiceCfg()

	if cfgPath != "" {
		if err := LoadCfg(cfgPath, serviceCfg); err != nil {
			abort("cannot load configuration: %v", err)
		}
	}

	daemonCfg, err := service.DaemonCfg()
	if err != nil {
		abort("invalid configuration: %v", err)
	}

	daemonCfg.name = name

	// Daemon
	d := newDaemon(daemonCfg, service)

	if err := d.init(); err != nil {
		abort("cannot initialize daemon: %v", err)
	}

	if err := d.start(); err != nil {
		abort("cannot start daemon: %v", err)
	}

	close(readyChan)

	d.wait()
	d.stop()

	d.terminate()
}
