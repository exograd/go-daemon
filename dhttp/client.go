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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/exograd/go-log"
)

type RequestError struct {
	Status   int
	APIError *APIError
	Message  string
}

func (err RequestError) Error() string {
	return err.Message
}

type ClientCfg struct {
	Log *log.Logger `json:"-"`

	LogRequests bool `json:"log_requests"`
}

type Client struct {
	Cfg ClientCfg
	Log *log.Logger

	client *http.Client
}

func NewClient(cfg ClientCfg) (*Client, error) {
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
		Transport: NewRoundTripper(transport, &cfg),
	}

	c := &Client{
		Cfg: cfg,
		Log: cfg.Log,

		client: client,
	}

	return c, nil
}

func (c *Client) Terminate() {
	c.client.CloseIdleConnections()
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

func (c *Client) SendRequest(method string, uri *url.URL, header map[string]string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, uri.String(), body)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %w")
	}

	for name, value := range header {
		req.Header.Set(name, value)
	}

	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		reqErr := &RequestError{
			Status:   res.StatusCode,
			APIError: nil,
			Message: fmt.Sprintf("request failed with status %d",
				res.StatusCode),
		}

		resBody, err := ioutil.ReadAll(res.Body)
		if err == nil {
			if res.Header.Get("Content-Type") == "application/json" {
				var apiErr APIError
				if err := json.Unmarshal(resBody, &apiErr); err == nil {
					reqErr.APIError = &apiErr
					reqErr.Message += ": " + apiErr.Message
				} else {
					c.Log.Error("cannot decode api error response: %w", err)
				}
			}

			if reqErr.APIError == nil && len(resBody) > 0 {
				reqErr.Message += ": " + string(resBody)
			}
		} else {
			c.Log.Error("cannot read response body: %w", err)
		}

		return res, reqErr
	}

	return res, nil
}

func (c *Client) SendJSONRequest(method string, uri *url.URL, header map[string]string, value interface{}) (*http.Response, error) {
	var body io.Reader

	if value != nil {
		var buf bytes.Buffer

		encoder := json.NewEncoder(&buf)
		if err := encoder.Encode(body); err != nil {
			return nil, fmt.Errorf("cannot encode request body: %w", err)
		}

		body = &buf
	}

	if header == nil {
		header = make(map[string]string)
	}

	if _, found := header["Content-Type"]; !found {
		header["Content-Type"] = "application/json"
	}

	return c.SendRequest(method, uri, header, body)
}
