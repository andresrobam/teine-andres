package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CredentialRepository struct {
	pool *pgxpool.Pool
}

func NewCredentialRepository(pool *pgxpool.Pool) *CredentialRepository {
	return &CredentialRepository{pool: pool}
}

func (r *CredentialRepository) GetSystemCredential(ctx context.Context, name string) (string, error) {
	if r.pool == nil {
		return "", errors.New("DATABASE_URL is not configured")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var secret string
	err := r.pool.QueryRow(dbCtx, "SELECT secret FROM system_credentials WHERE name = $1", name).Scan(&secret)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("system_credentials %s is required", name)
	}
	if err != nil {
		return "", err
	}

	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", fmt.Errorf("system_credentials %s is required", name)
	}

	return secret, nil
}
