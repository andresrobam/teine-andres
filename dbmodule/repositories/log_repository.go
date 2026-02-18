package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LogRepository struct {
	pool *pgxpool.Pool
}

func NewLogRepository(pool *pgxpool.Pool) *LogRepository {
	return &LogRepository{pool: pool}
}

func (r *LogRepository) InsertLog(ctx context.Context, eventType string, payload map[string]interface{}) error {
	if r.pool == nil {
		return errors.New("DATABASE_URL is not configured")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(dbCtx,
		"INSERT INTO log (id, event_type, payload, created_at) VALUES ($1, $2, $3, $4)",
		uuid.New(), eventType, payloadBytes, time.Now().UTC())

	return err
}
