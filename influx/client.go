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

package influx

import (
	"fmt"
	"sync"

	"github.com/exograd/go-daemon/dhttp"
	"github.com/exograd/go-log"
)

type ClientCfg struct {
	Log        *log.Logger   `json:"-"`
	HTTPClient *dhttp.Client `json:"-"`
	Hostname   string        `json:"-"`

	URI       string            `json:"uri"`
	Bucket    string            `json:"bucket"`
	Org       string            `json:"org"`
	BatchSize int               `json:"batchSize"`
	Precision Precision         `json:"precision"`
	Tags      map[string]string `json:"tags"`
}

func HTTPClientCfg(cfg *ClientCfg) dhttp.ClientCfg {
	return dhttp.ClientCfg{}
}

type Client struct {
	Cfg        ClientCfg
	Log        *log.Logger
	HTTPClient *dhttp.Client

	tags map[string]string

	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewClient(cfg ClientCfg) (*Client, error) {
	if cfg.Log == nil {
		cfg.Log = log.DefaultLogger("influx")
	}

	cfg.HTTPClient = cfg.HTTPClient

	if cfg.URI == "" {
		cfg.URI = "http://localhost:8086"
	}

	if cfg.Bucket == "" {
		return nil, fmt.Errorf("missing or empty bucket")
	}

	if cfg.BatchSize == 0 {
		cfg.BatchSize = 10_000
	}

	tags := make(map[string]string)
	tags["host"] = cfg.Hostname
	for name, value := range cfg.Tags {
		tags[name] = value
	}

	c := &Client{
		Cfg: cfg,
		Log: cfg.Log,

		stopChan: make(chan struct{}),
	}

	return c, nil
}

func (c *Client) Start() {
	c.wg.Add(1)
	go c.main()
}

func (c *Client) Stop() {
	close(c.stopChan)
	c.wg.Wait()
}

func (c *Client) Terminate() {
}

func (c *Client) main() {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopChan:
			return
		}
	}
}
