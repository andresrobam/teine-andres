package dbmodule

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool = pgxpool.Pool

func NewPool(ctx context.Context, databaseURL string) (*Pool, func(), error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, nil, nil
	}

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(dbCtx, databaseURL)
	if err != nil {
		return nil, nil, err
	}

	return pool, pool.Close, nil
}

func Read(ctx context.Context, pool *Pool, query string) (string, error) {
	if pool == nil {
		return "", errors.New("DATABASE_URL is not configured")
	}

	q := strings.TrimSpace(query)
	if q == "" {
		return "", errors.New("query is required")
	}
	if !isReadQuery(q) {
		return "", errors.New("read tool only supports SELECT/CTE/SHOW/DESCRIBE/EXPLAIN queries")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	rows, err := pool.Query(dbCtx, q)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	cols := rows.FieldDescriptions()
	var results []map[string]interface{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return "", err
		}
		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[string(col.Name)] = string(b)
				continue
			}
			row[string(col.Name)] = val
		}
		results = append(results, row)
	}
	if rows.Err() != nil {
		return "", rows.Err()
	}

	payload, err := json.Marshal(map[string]interface{}{
		"rows": results,
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func Modify(ctx context.Context, pool *Pool, query string) (string, error) {
	if pool == nil {
		return "", errors.New("DATABASE_URL is not configured")
	}

	q := strings.TrimSpace(query)
	if q == "" {
		return "", errors.New("query is required")
	}
	if isReadQuery(q) {
		return "", errors.New("modify tool does not support SELECT/CTE/SHOW/DESCRIBE/EXPLAIN queries")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	tag, err := pool.Exec(dbCtx, q)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(map[string]interface{}{
		"rows_affected": tag.RowsAffected(),
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func isReadQuery(query string) bool {
	upper := strings.ToUpper(strings.TrimSpace(query))
	return strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "WITH") ||
		strings.HasPrefix(upper, "SHOW") || strings.HasPrefix(upper, "DESCRIBE") ||
		strings.HasPrefix(upper, "EXPLAIN")
}
