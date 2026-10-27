package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/ivancabzon/task-service/internal/model"
	"github.com/ivancabzon/task-service/pkg/apperr"
)

type TaskRepo struct {
	db *sqlx.DB
}

func NewTaskRepo(db *sqlx.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) Create(ctx context.Context, t *model.Task) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id, closed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.TeamID, t.Title, t.Description, t.Status, t.CreatedBy, t.AssigneeID, t.ClosedAt)
	if err != nil {
		return 0, fmt.Errorf("insert task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("task last insert id: %w", err)
	}
	return id, nil
}

func (r *TaskRepo) GetByID(ctx context.Context, id int64) (*model.Task, error) {
	var t model.Task
	err := r.db.GetContext(ctx, &t, `SELECT * FROM tasks WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.NotFound("task not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	return &t, nil
}

func (r *TaskRepo) List(ctx context.Context, f model.TaskFilter) ([]model.Task, error) {
	q := `SELECT * FROM tasks WHERE team_id = ?`
	args := []any{f.TeamID}

	if f.Status != "" {
		q += ` AND status = ?`
		args = append(args, f.Status)
	}
	if f.AssigneeID != nil {
		q += ` AND assignee_id = ?`
		args = append(args, *f.AssigneeID)
	}
	q += ` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
	args = append(args, f.Limit, f.Offset)

	tasks := make([]model.Task, 0)
	if err := r.db.SelectContext(ctx, &tasks, q, args...); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

// UpdateWithHistory — ОДНА транзакция:
//  1. UPDATE с проверкой version (optimistic lock)
//  2. INSERT в task_history
//     Если version не совпала → apperr.Conflict, транзакция откатывается.
func (r *TaskRepo) UpdateWithHistory(
	ctx context.Context,
	t *model.Task,
	changes model.ChangesMap,
	userID int64,
) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. UPDATE с проверкой version
	res, err := tx.ExecContext(ctx, `
		UPDATE tasks SET
			title = ?,
			description = ?,
			status = ?,
			assignee_id = ?,
			closed_at = ?,
			version = version + 1,
			updated_at = NOW()
		WHERE id = ? AND version = ?`,
		t.Title, t.Description, t.Status, t.AssigneeID, t.ClosedAt, t.ID, t.Version)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return apperr.Conflict("task was modified by another user; reload and retry")
	}

	// 2. История
	if len(changes) > 0 {
		changesJSON, err := json.Marshal(changes)
		if err != nil {
			return fmt.Errorf("marshal changes: %w", err)
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO task_history (task_id, changed_by, changes)
			VALUES (?, ?, ?)`,
			t.ID, userID, changesJSON)
		if err != nil {
			return fmt.Errorf("insert history: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit task update: %w", err)
	}
	return nil
}
