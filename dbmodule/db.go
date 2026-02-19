package dbmodule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool = pgxpool.Pool

// DualPool holds both agent and owner database connection pools
type DualPool struct {
	Agent *Pool
	Owner *Pool
}

// buildConnectionString constructs a full PostgreSQL connection string
// by injecting the username and password into a base URL
func buildConnectionString(baseURL, username, password string) (string, error) {
	if strings.TrimSpace(baseURL) == "" {
		return "", errors.New("base database URL is required")
	}
	if strings.TrimSpace(password) == "" {
		return "", fmt.Errorf("password for user %s is required", username)
	}

	// Parse the base URL
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid database URL: %w", err)
	}

	// Set user info with hardcoded username and provided password
	u.User = url.UserPassword(username, password)

	return u.String(), nil
}

// NewPool creates two separate connection pools: one for the agent user and one for the owner user
// baseURL: The database URL without credentials (e.g., "postgres://localhost:5432/teine?sslmode=disable")
// agentPassword: Password for the 'agent' user
// ownerPassword: Password for the 'owner' user
func NewPool(ctx context.Context, baseURL, agentPassword, ownerPassword string) (*DualPool, func(), error) {
	// Build connection strings with hardcoded usernames
	agentURL, err := buildConnectionString(baseURL, "agent", agentPassword)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build agent connection string: %w", err)
	}

	ownerURL, err := buildConnectionString(baseURL, "owner", ownerPassword)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build owner connection string: %w", err)
	}

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Create agent pool
	agentPool, err := pgxpool.New(dbCtx, agentURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create agent pool: %w", err)
	}

	// Create owner pool
	ownerPool, err := pgxpool.New(dbCtx, ownerURL)
	if err != nil {
		agentPool.Close()
		return nil, nil, fmt.Errorf("failed to create owner pool: %w", err)
	}

	dualPool := &DualPool{
		Agent: agentPool,
		Owner: ownerPool,
	}

	cleanup := func() {
		agentPool.Close()
		ownerPool.Close()
	}

	return dualPool, cleanup, nil
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
