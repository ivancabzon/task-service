package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/ivancabzon/task-service/internal/model"
)

type HistoryRepo struct {
	db *sqlx.DB
}

func NewHistoryRepo(db *sqlx.DB) *HistoryRepo {
	return &HistoryRepo{db: db}
}

// история задачи, от старых к новым.
func (r *HistoryRepo) ListByTask(ctx context.Context, taskID int64) ([]model.TaskHistory, error) {
	rows := make([]model.TaskHistory, 0)
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, task_id, changed_by, changes, created_at
		FROM task_history
		WHERE task_id = ?
		ORDER BY created_at ASC, id ASC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list task history: %w", err)
	}
	return rows, nil
}
