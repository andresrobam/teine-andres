package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SyncStateRepository struct {
	pool *pgxpool.Pool
}

func NewSyncStateRepository(pool *pgxpool.Pool) *SyncStateRepository {
	return &SyncStateRepository{pool: pool}
}

func (r *SyncStateRepository) GetNextBatch(ctx context.Context) (string, error) {
	if r.pool == nil {
		return "", nil
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var nextBatch *string
	err := r.pool.QueryRow(dbCtx, "SELECT next_batch FROM matrix_sync_state LIMIT 1").Scan(&nextBatch)
	if err != nil {
		return "", err
	}

	if nextBatch == nil {
		return "", nil
	}
	return *nextBatch, nil
}

func (r *SyncStateRepository) UpdateNextBatch(ctx context.Context, token string) error {
	if r.pool == nil {
		return nil
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.pool.Exec(dbCtx, "UPDATE matrix_sync_state SET next_batch = $1, updated_at = now()", token)
	return err
}
