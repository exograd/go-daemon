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
	"os"
	"path"
	"sort"
	"time"
)

const MigrationVersionLayout = "20060102T150405Z"

type Migration struct {
	Schema  string
	Version string
	Code    []byte
}

type Migrations []*Migration

func (m *Migration) String() string {
	return fmt.Sprintf("%s-%s", m.Schema, m.Version)
}

func (m *Migration) LoadFile(filePath string) error {
	baseName := path.Base(filePath)
	ext := path.Ext(baseName)
	baseName = baseName[:len(baseName)-len(ext)]

	if err := ValidateMigrationVersion(baseName); err != nil {
		return fmt.Errorf("invalid migration version %q: invalid format",
			baseName)
	}

	code, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read %q: %w", filePath, err)
	}

	m.Version = baseName
	m.Code = code

	return nil
}

func (m *Migration) Apply(conn Conn) error {
	ctx := context.Background()

	if _, err := conn.Exec(ctx, string(m.Code)); err != nil {
		return fmt.Errorf("cannot execute migration: %w", err)
	}

	query := `
INSERT INTO schema_versions (schema, version)
  VALUES ($1, $2)
`
	if _, err := conn.Exec(ctx, query, m.Schema, m.Version); err != nil {
		return fmt.Errorf("cannot insert schema version: %w", err)
	}

	return nil
}

func (pms *Migrations) LoadDirectory(schema, dirPath string) error {
	var ms Migrations

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("cannot read directory %q: %w", dirPath, err)
	}

	for _, e := range entries {
		name := e.Name()

		ext := path.Ext(name)
		if ext != ".sql" {
			continue
		}

		filePath := path.Join(dirPath, name)

		var m Migration
		if err := m.LoadFile(filePath); err != nil {
			return fmt.Errorf("cannot load migration from %q: %w",
				filePath, err)
		}

		m.Schema = schema

		ms = append(ms, &m)
	}

	*pms = ms
	return nil
}

func (ms Migrations) Sort() {
	sort.Slice(ms, func(i, j int) bool {
		return ms[i].Version < ms[j].Version
	})
}

func (pms *Migrations) RejectVersions(versions map[string]struct{}) {
	ms := *pms

	var ms2 Migrations
	for _, m := range ms {
		if _, found := versions[m.Version]; !found {
			ms2 = append(ms2, m)
		}
	}

	*pms = ms2
}

func ValidateMigrationVersion(s string) (err error) {
	_, err = time.Parse(MigrationVersionLayout, s)
	return
}
