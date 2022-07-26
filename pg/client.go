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
	"path"

	"github.com/exograd/go-daemon/check"
	"github.com/exograd/go-daemon/dlog"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Advisory locks are identified by two 32 bit integers. We arbitrarily
// reserve a value for the first one for all locks taken by go-daemon.
const AdvisoryLockId1 uint32 = 0x00ff

const (
	AdvisoryLockId2Migrations uint32 = 0x0001
)

type ClientCfg struct {
	Log *dlog.Logger `json:"-"`

	URI string `json:"uri"`

	SchemaDirectory string   `json:"schema_directory"`
	SchemaNames     []string `json:"schema_names"`
}

func (cfg *ClientCfg) Check(c *check.Checker) {
	c.CheckStringURI("uri", cfg.URI)

	c.CheckStringNotEmpty("schema_directory", cfg.SchemaDirectory)

	c.WithChild("schema_names", func() {
		for i, name := range cfg.SchemaNames {
			c.CheckStringNotEmpty(i, name)
		}
	})
}

type Client struct {
	Cfg ClientCfg
	Log *dlog.Logger

	Pool *pgxpool.Pool
}

func NewClient(cfg ClientCfg) (*Client, error) {
	if cfg.Log == nil {
		cfg.Log = dlog.DefaultLogger("pg")
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

	if c.Cfg.SchemaDirectory != "" {
		if err := c.updateSchemas(); err != nil {
			c.Close()
			return nil, err
		}
	}

	return c, nil
}

func (c *Client) updateSchemas() error {
	for _, name := range c.Cfg.SchemaNames {
		dirPath := path.Join(c.Cfg.SchemaDirectory, name)

		if err := c.UpdateSchema(name, dirPath); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) Close() {
	c.Pool.Close()
}

func (c *Client) WithConn(fn func(Conn) error) error {
	ctx := context.Background()

	conn, err := c.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("cannot acquire connection: %w", err)
	}
	defer conn.Release()

	return fn(conn)
}

func (c *Client) WithTx(fn func(Conn) error) (err error) {
	ctx := context.Background()

	conn, acquireErr := c.Pool.Acquire(ctx)
	if acquireErr != nil {
		err = fmt.Errorf("cannot acquire connection: %w", acquireErr)
		return
	}
	defer conn.Release()

	if _, beginErr := conn.Exec(ctx, "BEGIN"); beginErr != nil {
		err = fmt.Errorf("cannot begin transaction: %w", beginErr)
		return
	}

	defer func() {
		if err != nil {
			// If an error was already signaled, do not commit
			return
		}

		if _, commitErr := conn.Exec(ctx, "COMMIT"); commitErr != nil {
			err = fmt.Errorf("cannot commit transaction: %w", commitErr)
		}
	}()

	if fnErr := fn(conn); fnErr != nil {
		err = fnErr

		if _, rollbackErr := conn.Exec(ctx, "ROLLBACK"); rollbackErr != nil {
			// There is nothing we can do here, and we do want to return the
			// function error, so we simply log the rollback error.
			c.Log.Error("cannot rollback transaction: %v", err)
		}
	}

	return
}

func (c *Client) UpdateSchema(schema, dirPath string) error {
	c.Log.Info("updating schema %q using migrations from %q", schema, dirPath)

	var migrations Migrations
	if err := migrations.LoadDirectory(schema, dirPath); err != nil {
		return fmt.Errorf("cannot load migrations: %w", err)
	}

	if len(migrations) == 0 {
		c.Log.Info("no migration available")
		return nil
	}

	err := c.WithTx(func(conn Conn) error {
		// Take a lock to make sure only one application tries to update the
		// schema at the same time.
		err := TakeAdvisoryLock(conn,
			AdvisoryLockId1, AdvisoryLockId2Migrations)
		if err != nil {
			return fmt.Errorf("cannot take advisory lock: %w", err)
		}

		// Create the table if it does not exist. Note that we do not use the
		// current connection because we need each migration, which will be
		// executed in its own transaction (i.e. before the the end of the
		// main transaction), to see it.
		if err := c.WithConn(createSchemaVersionTable); err != nil {
			return fmt.Errorf("cannot create schema version table: %w", err)
		}

		// Load currently applied versions and remove them from the set of
		// migrations.
		appliedVersions, err := loadSchemaVersions(conn, schema)
		if err != nil {
			return fmt.Errorf("cannot load schema versions: %w", err)
		}

		migrations.RejectVersions(appliedVersions)

		// Apply migrations in order
		migrations.Sort()

		for _, m := range migrations {
			c.Log.Info("applying migration %v", m)

			if err := c.WithTx(m.Apply); err != nil {
				return fmt.Errorf("cannot apply migration %v: %w", m, err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Close connections in case migrations created new types;
	// this way these types will be discovered by pgx during the next
	// connections.
	ctx := context.Background()
	conns := c.Pool.AcquireAllIdle(ctx)
	for _, conn := range conns {
		conn.Conn().Close(ctx)
		conn.Release()
	}

	return nil
}

func TakeAdvisoryLock(conn Conn, id1, id2 uint32) error {
	ctx := context.Background()

	query := `SELECT pg_advisory_xact_lock($1, $2)`
	_, err := conn.Exec(ctx, query, id1, id2)
	return err
}

func createSchemaVersionTable(conn Conn) error {
	ctx := context.Background()

	query := `
CREATE TABLE IF NOT EXISTS schema_versions
  (schema VARCHAR NOT NULL,
   version VARCHAR NOT NULL,
   migration_date TIMESTAMP NOT NULL
     DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),

   PRIMARY KEY (schema, version)
)
`
	_, err := conn.Exec(ctx, query)
	return err
}

func loadSchemaVersions(conn Conn, schema string) (map[string]struct{}, error) {
	ctx := context.Background()

	query := `SELECT version FROM schema_versions WHERE schema = $1`
	rows, err := conn.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make(map[string]struct{})

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}

		versions[version] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return versions, nil
}
