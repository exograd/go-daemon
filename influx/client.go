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
	"time"

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

	pointsChan chan Points
	points     Points

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

		pointsChan: make(chan Points),

		stopChan: make(chan struct{}),
	}

	return c, nil
}

func (c *Client) Start() {
	c.wg.Add(1)
	go c.main()

	c.wg.Add(1)
	go c.goProbeMain()
}

func (c *Client) Stop() {
	close(c.stopChan)
	c.wg.Wait()
}

func (c *Client) Terminate() {
	close(c.pointsChan)
}

func (c *Client) main() {
	defer c.wg.Done()

	timer := time.NewTicker(time.Second)
	defer timer.Stop()

	for {
		select {
		case <-c.stopChan:
			c.flush()
			return

		case ps := <-c.pointsChan:
			c.enqueuePoints(ps)

		case <-timer.C:
			c.flush()
		}
	}
}

func (c *Client) EnqueuePoint(p *Point) {
	c.EnqueuePoints(Points{p})
}

func (c *Client) EnqueuePoints(ps Points) {
	c.pointsChan <- ps
}

func (c *Client) enqueuePoints(ps Points) {
	c.points = append(c.points, ps...)

	if len(c.points) >= c.Cfg.BatchSize {
		c.flush()
	}
}

func (c *Client) flush() {
	if len(c.points) == 0 {
		return
	}

	// TODO
	c.Log.Info("flushing %d points", len(c.points))
	c.points = nil
}
