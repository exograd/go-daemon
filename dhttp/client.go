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
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/dlog"
)

type ClientCfg struct {
	Log *dlog.Logger `json:"-"`

	LogRequests bool `json:"log_requests"`

	TLS *TLSClientCfg `json:"tls"`

	Header http.Header `json:"-"`
}

type TLSClientCfg struct {
	CACertificates []string            `json:"ca_certificates"`
	PublicKeyPins  map[string][]string `json:"public_key_pins"`
}

type Client struct {
	Cfg ClientCfg
	Log *dlog.Logger

	Client *http.Client

	tlsCfg *tls.Config
}

func (cfg *ClientCfg) Check(c *check.Checker) {
	c.CheckOptionalObject("tls", cfg.TLS)
}

func (cfg *TLSClientCfg) Check(c *check.Checker) {
	c.WithChild("ca_certificates", func() {
		for i, cert := range cfg.CACertificates {
			c.CheckStringNotEmpty(i, cert)
		}
	})

	c.WithChild("public_key_pins", func() {
		for serverName, pins := range cfg.PublicKeyPins {
			c.WithChild(serverName, func() {
				for i, pin := range pins {
					c.CheckStringNotEmpty(i, pin)
				}
			})
		}
	})
}

func NewClient(cfg ClientCfg) (*Client, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,

		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,

		MaxIdleConns: 100,

		IdleConnTimeout:       60 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	tlsCfg := &tls.Config{}

	if cfg.TLS != nil {
		caCertificatePool, err := LoadCertificates(cfg.TLS.CACertificates)
		if err != nil {
			return nil, err
		}

		tlsCfg.RootCAs = caCertificatePool
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: NewRoundTripper(transport, &cfg),
	}

	c := &Client{
		Cfg: cfg,
		Log: cfg.Log,

		Client: client,

		tlsCfg: tlsCfg,
	}

	transport.DialTLSContext = c.DialTLSContext

	return c, nil
}

func (c *Client) Terminate() {
	c.Client.CloseIdleConnections()
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.Client.Do(req)
}

func (c *Client) SendRequest(method string, uri *url.URL, header map[string]string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, uri.String(), body)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %w", err)
	}

	for name, value := range header {
		req.Header.Set(name, value)
	}

	return c.Do(req)
}

func (c *Client) DialTLSContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
		Config: c.tlsCfg,
	}

	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	if err := c.checkTLSPublicKey(conn.(*tls.Conn)); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func (c *Client) checkTLSPublicKey(conn *tls.Conn) error {
	if c.Cfg.TLS == nil {
		return nil
	}

	state := conn.ConnectionState()

	pins, found := c.Cfg.TLS.PublicKeyPins[state.ServerName]
	if !found || len(pins) == 0 {
		return nil
	}

	if len(state.PeerCertificates) == 0 {
		return fmt.Errorf("no peer certificate available")
	}

	cert := state.PeerCertificates[0]
	pubKeyData, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return fmt.Errorf("cannot marshal public key: %w", err)
	}

	hash := sha256.Sum256(pubKeyData)
	hexHash := hex.EncodeToString(hash[:])

	found = false
	for _, pin := range pins {
		if hexHash == pin {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("invalid server certificate: unknown public key")
	}

	return nil
}
