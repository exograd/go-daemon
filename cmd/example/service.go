package main

import (
	"github.com/exograd/go-daemon/daemon"
	"github.com/exograd/go-daemon/dhttp"
	"github.com/exograd/go-daemon/influx"
	"github.com/exograd/go-log"
)

type ServiceCfg struct {
}

type Service struct {
	Cfg ServiceCfg

	Daemon *daemon.Daemon
	Log    *log.Logger
}

func NewService() *Service {
	s := &Service{}

	return s
}

func (s *Service) ServiceCfg() interface{} {
	cfg := ServiceCfg{}

	s.Cfg = cfg

	return &s.Cfg
}

func (s *Service) DaemonCfg() (daemon.DaemonCfg, error) {
	cfg := daemon.NewDaemonCfg()

	cfg.HTTPServers["main"] = dhttp.ServerCfg{
		Address: "localhost:8080",
	}

	cfg.HTTPClients["default"] = dhttp.ClientCfg{}

	cfg.Influx = &influx.ClientCfg{
		Bucket: "daemon/main",
	}

	return cfg, nil
}

func (s *Service) Init(d *daemon.Daemon) error {
	s.Daemon = d
	s.Log = d.Log

	return nil
}

func (s *Service) Start(d *daemon.Daemon) error {
	return nil
}

func (s *Service) Stop(d *daemon.Daemon) {
}

func (s *Service) Terminate(d *daemon.Daemon) {
}
