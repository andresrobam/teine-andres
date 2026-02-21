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

	rows, err := r.pool.Query(dbCtx, "SELECT title, prompt, load_order FROM identity ORDER BY load_order ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []models.Prompt
	for rows.Next() {
		var p models.Prompt
		if err := rows.Scan(&p.Title, &p.Prompt, &p.LoadOrder); err != nil {
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

	rows, err := r.pool.Query(dbCtx, "SELECT title, prompt, load_order FROM self ORDER BY load_order ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []models.Prompt
	for rows.Next() {
		var p models.Prompt
		if err := rows.Scan(&p.Title, &p.Prompt, &p.LoadOrder); err != nil {
			return nil, err
		}
		prompts = append(prompts, p)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return prompts, nil
}

func (r *PromptRepository) UpsertIdentityPrompt(ctx context.Context, title string, prompt string) error {
	if r.pool == nil {
		return errors.New("DATABASE_URL is not configured")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	query := `
		INSERT INTO identity (title, prompt, load_order)
		VALUES ($1, $2, COALESCE((SELECT MAX(load_order) FROM identity), 0) + 1)
		ON CONFLICT (title) DO UPDATE SET prompt = EXCLUDED.prompt
	`

	_, err := r.pool.Exec(dbCtx, query, title, prompt)
	return err
}
