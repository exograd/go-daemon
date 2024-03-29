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
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/dhttp"
	"github.com/exograd/go-daemon/dlog"
)

type ClientCfg struct {
	Log        *dlog.Logger  `json:"-"`
	HTTPClient *dhttp.Client `json:"-"`
	Hostname   string        `json:"-"`

	URI         string            `json:"uri"`
	Bucket      string            `json:"bucket"`
	Org         string            `json:"org"`
	BatchSize   int               `json:"batch_size"`
	Tags        map[string]string `json:"tags"`
	LogRequests bool              `json:"log_requests"`
}

func (cfg *ClientCfg) Check(c *check.Checker) {
	// The organization is optional (it is only used for InfluxDB 2.x)

	c.CheckStringURI("uri", cfg.URI)
	c.CheckStringNotEmpty("bucket", cfg.Bucket)

	if cfg.BatchSize != 0 {
		c.CheckIntMin("batch_size", cfg.BatchSize, 1)
	}

	c.WithChild("tags", func() {
		for name, value := range cfg.Tags {
			c.CheckStringNotEmpty(name, value)
		}
	})
}

func HTTPClientCfg(cfg *ClientCfg) dhttp.ClientCfg {
	return dhttp.ClientCfg{
		LogRequests: cfg.LogRequests,
	}
}

type Client struct {
	Cfg        ClientCfg
	Log        *dlog.Logger
	HTTPClient *dhttp.Client

	uri  *url.URL
	tags map[string]string

	pointsChan chan Points
	points     Points

	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewClient(cfg ClientCfg) (*Client, error) {
	if cfg.Log == nil {
		cfg.Log = dlog.DefaultLogger("influx")
	}

	if cfg.HTTPClient == nil {
		return nil, fmt.Errorf("missing http client")
	}

	if cfg.URI == "" {
		cfg.URI = "http://localhost:8086"
	}
	uri, err := url.Parse(cfg.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid uri: %w", err)
	}

	if cfg.Bucket == "" {
		return nil, fmt.Errorf("missing or empty bucket")
	}

	if cfg.BatchSize == 0 {
		cfg.BatchSize = 10_000
	}

	tags := make(map[string]string)
	if cfg.Hostname != "" {
		tags["host"] = cfg.Hostname
	}
	for name, value := range cfg.Tags {
		tags[name] = value
	}

	c := &Client{
		Cfg:        cfg,
		Log:        cfg.Log,
		HTTPClient: cfg.HTTPClient,

		uri:  uri,
		tags: tags,

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

func (c *Client) EnqueuePoints(points Points) {
	// We do not want to be stuck writing on c.pointsChan if the server is
	// stopping, so we check the stop chan.

	select {
	case <-c.stopChan:
		return

	case c.pointsChan <- points:
	}
}

func (c *Client) enqueuePoints(points Points) {
	for _, p := range points {
		c.finalizePoint(p)
	}

	c.points = append(c.points, points...)

	if len(c.points) >= c.Cfg.BatchSize {
		c.flush()
	}
}

func (c *Client) finalizePoint(point *Point) {
	tags := Tags{}

	for key, value := range c.tags {
		if value != "" {
			tags[key] = value
		}
	}

	for key, value := range point.Tags {
		if value != "" {
			tags[key] = value
		}
	}

	point.Tags = tags
}

func (c *Client) flush() {
	if len(c.points) == 0 {
		return
	}

	if err := c.sendPoints(c.points); err != nil {
		c.Log.Error("cannot send points: %v", err)
		return
	}

	c.points = nil
}

func (c *Client) sendPoints(points Points) error {
	uri := *c.uri
	uri.Path = path.Join(uri.Path, "/api/v2/write")

	query := url.Values{}
	query.Set("bucket", c.Cfg.Bucket)
	if c.Cfg.Org != "" {
		query.Set("org", c.Cfg.Org)
	}

	uri.RawQuery = query.Encode()

	var buf bytes.Buffer
	EncodePoints(points, &buf)

	req, err := http.NewRequest("POST", uri.String(), &buf)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot send request: %w", err)
	}
	defer res.Body.Close()

	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		bodyData, err := ioutil.ReadAll(res.Body)
		if err != nil {
			c.Log.Error("cannot read response body: %v", err)
		}

		bodyString := ""
		if bodyData != nil {
			// Influx can send incredibly long error messages, sometimes
			// including the entire payload received. This is very annoying,
			// but even if it was to be patched, we would still have to
			// support old versions.
			if len(bodyData) > 200 {
				bodyData = append(bodyData[:200], []byte(" [truncated]")...)
			}

			bodyString = " (" + string(bodyData) + ")"
		}

		return fmt.Errorf("request failed with status %d%s",
			res.StatusCode, bodyString)
	}

	return nil
}
