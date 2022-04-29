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

package pg

import (
	"context"
	"fmt"

	"github.com/exograd/go-log"
	"github.com/jackc/pgx/v4/pgxpool"
)

type ClientCfg struct {
	Log *log.Logger `json:"-"`

	URI string `json:"uri"`
}

type Client struct {
	Cfg ClientCfg
	Log *log.Logger

	Pool *pgxpool.Pool
}

func NewClient(cfg ClientCfg) (*Client, error) {
	if cfg.Log == nil {
		cfg.Log = log.DefaultLogger("pg")
	}

	if cfg.URI == "" {
		return nil, fmt.Errorf("missing or empty url")
	}

	cfg.Log.Info("connecting to %q", cfg.URI)

	poolCfg, err := pgxpool.ParseConfig(cfg.URI)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.ConnectConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to database at %q: %w",
			cfg.URI, err)
	}

	c := &Client{
		Cfg: cfg,
		Log: cfg.Log,

		Pool: pool,
	}

	return c, nil
}

func (c *Client) Close() {
	c.Pool.Close()
}
