package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"teine-andres/dbmodule/models"
)

type PromptRepository struct {
	pool *pgxpool.Pool
}

func NewPromptRepository(pool *pgxpool.Pool) *PromptRepository {
	return &PromptRepository{pool: pool}
}

func (r *PromptRepository) GetIdentityPrompts(ctx context.Context) ([]models.Prompt, error) {
	if r.pool == nil {
		return nil, errors.New("DATABASE_URL is not configured")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.pool.Query(dbCtx, "SELECT id, title, prompt, load_order, NULL as tags, created_at, NULL as updated_at FROM identity ORDER BY load_order ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []models.Prompt
	for rows.Next() {
		var p models.Prompt
		if err := rows.Scan(&p.ID, &p.Title, &p.Prompt, &p.LoadOrder, &p.Tags, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prompts = append(prompts, p)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return prompts, nil
}

func (r *PromptRepository) GetSelfPrompts(ctx context.Context) ([]models.Prompt, error) {
	if r.pool == nil {
		return nil, errors.New("DATABASE_URL is not configured")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.pool.Query(dbCtx, "SELECT id, title, prompt, load_order, tags, created_at, updated_at FROM self ORDER BY load_order ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []models.Prompt
	for rows.Next() {
		var p models.Prompt
		if err := rows.Scan(&p.ID, &p.Title, &p.Prompt, &p.LoadOrder, &p.Tags, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prompts = append(prompts, p)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return prompts, nil
}
