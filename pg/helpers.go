package pg

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
)

type Object interface {
	FromRow(pgx.Row) error
}

type Objects interface {
	AddFromRow(pgx.Row) error
}

func Exec(conn Conn, query string, args ...interface{}) error {
	ctx := context.Background()
	return ExecContext(ctx, conn, query, args...)
}

func ExecContext(ctx context.Context, conn Conn, query string, args ...interface{}) error {
	_, err := conn.Exec(ctx, query, args...)
	return err
}

func Exec2(conn Conn, query string, args ...interface{}) (int64, error) {
	ctx := context.Background()
	return Exec2Context(ctx, conn, query, args...)
}

func Exec2Context(ctx context.Context, conn Conn, query string, args ...interface{}) (int64, error) {
	tag, err := conn.Exec(ctx, query, args...)
	if err != nil {
		return -1, err
	}

	return tag.RowsAffected(), nil
}

func QueryObject(conn Conn, obj Object, query string, args ...interface{}) error {
	ctx := context.Background()
	return QueryObjectContext(ctx, conn, obj, query, args...)
}

func QueryObjectContext(ctx context.Context, conn Conn, obj Object, query string, args ...interface{}) error {
	row := conn.QueryRow(ctx, query, args...)
	return obj.FromRow(row)
}

func QueryObjects(conn Conn, objs Objects, query string, args ...interface{}) error {
	ctx := context.Background()
	return QueryObjectsContext(ctx, conn, objs, query, args...)
}

func QueryObjectsContext(ctx context.Context, conn Conn, objs Objects, query string, args ...interface{}) error {
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("cannot execute query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if err := objs.AddFromRow(rows); err != nil {
			return fmt.Errorf("cannot read row: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("cannot read query response: %w", err)
	}

	return nil
}
