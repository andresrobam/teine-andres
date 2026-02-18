package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"teine-andres/dbmodule/models"
)

type TaskRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

func (r *TaskRepository) GetTasksByStatuses(ctx context.Context, statuses []string) ([]models.Task, error) {
	if r.pool == nil {
		return nil, errors.New("DATABASE_URL is not configured")
	}

	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			t.id, 
			t.parent_task_id, 
			t.title, 
			t.description, 
			t.created_at,
			ts.status,
			ts.progress,
			ts.updated_at as status_updated_at
		FROM tasks t
		LEFT JOIN task_status ts ON t.id = ts.task_id
		WHERE ts.status = ANY($1)
		ORDER BY t.created_at DESC
	`

	rows, err := r.pool.Query(dbCtx, query, statuses)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.ParentID, &t.Title, &t.Description, &t.CreatedAt, &t.Status, &t.Progress, &t.StatusUpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return tasks, nil
}
