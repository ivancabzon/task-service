package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/ivancabzon/task-service/internal/model"
)

type CommentRepo struct {
	db *sqlx.DB
}

func NewCommentRepo(db *sqlx.DB) *CommentRepo {
	return &CommentRepo{db: db}
}

func (r *CommentRepo) Create(ctx context.Context, c *model.TaskComment) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO task_comments (task_id, user_id, content)
		VALUES (?, ?, ?)`, c.TaskID, c.UserID, c.Content)
	if err != nil {
		return 0, fmt.Errorf("insert comment: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("comment last insert id: %w", err)
	}
	return id, nil
}

func (r *CommentRepo) GetByID(ctx context.Context, id int64) (*model.TaskComment, error) {
	var c model.TaskComment
	err := r.db.GetContext(ctx, &c, `SELECT * FROM task_comments WHERE id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("get comment: %w", err)
	}
	return &c, nil
}

func (r *CommentRepo) ListByTask(ctx context.Context, taskID int64) ([]model.TaskComment, error) {
	rows := make([]model.TaskComment, 0)
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, task_id, user_id, content, created_at
		FROM task_comments
		WHERE task_id = ?
		ORDER BY created_at ASC, id ASC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	return rows, nil
}
