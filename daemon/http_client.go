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
	"net"
	"net/http"
	"time"

	"github.com/exograd/go-log"
)

type HTTPClientCfg struct {
	Log *log.Logger `json:"-"`
}

type HTTPClient struct {
	Cfg HTTPClientCfg
	Log *log.Logger

	client *http.Client
}

func NewHTTPClient(cfg HTTPClientCfg) (*HTTPClient, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: NewHTTPRoundTripper(transport, cfg.Log),
	}

	c := &HTTPClient{
		Cfg: cfg,
		Log: cfg.Log,

		client: client,
	}

	return c, nil
}

func (c *HTTPClient) Terminate() {
	c.client.CloseIdleConnections()
}

func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}
