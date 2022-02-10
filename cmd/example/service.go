package main

import (
	"github.com/exograd/go-daemon/daemon"
	"github.com/exograd/go-log"
)

type ServiceCfg struct {
	Logger        *log.LoggerCfg       `json:"logger"`
	HTTPServerCfg daemon.HTTPServerCfg `json:"httpServer"`
	HTTPClientCfg daemon.HTTPClientCfg `json:"httpClient"`
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
	cfg := daemon.DaemonCfg{
		Logger: s.Cfg.Logger,

		HTTPServers: make(map[string]daemon.HTTPServerCfg),
		HTTPClients: make(map[string]daemon.HTTPClientCfg),
	}

	cfg.HTTPServers["main"] = s.Cfg.HTTPServerCfg
	cfg.HTTPClients["default"] = s.Cfg.HTTPClientCfg

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
