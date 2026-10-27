package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/ivancabzon/task-service/internal/model"
)

type StatsRepo struct {
	db *sqlx.DB
}

func NewStatsRepo(db *sqlx.DB) *StatsRepo {
	return &StatsRepo{db: db}
}

func (r *StatsRepo) GetTeamStats(ctx context.Context, teamID int64) (*model.TeamStatsResponse, error) {
	const q = `
WITH
    status_counts AS (
        SELECT status, COUNT(*) AS cnt
        FROM tasks
        WHERE team_id = ?
        GROUP BY status
    ),
    top_assignees AS (
        SELECT assignee_id, COUNT(*) AS closed_count
        FROM tasks
        WHERE team_id = ?
          AND status = 'closed'
          AND assignee_id IS NOT NULL
          AND closed_at >= NOW() - INTERVAL 30 DAY
        GROUP BY assignee_id
        ORDER BY closed_count DESC, assignee_id ASC
        LIMIT 3
    ),
    avg_close AS (
        SELECT AVG(TIMESTAMPDIFF(SECOND, created_at, closed_at)) AS avg_sec
        FROM tasks
        WHERE team_id = ?
          AND status = 'closed'
          AND closed_at IS NOT NULL
    ),
    comment_counts AS (
        SELECT t.id AS task_id, COUNT(tc.id) AS cnt
        FROM tasks t
        LEFT JOIN task_comments tc ON tc.task_id = t.id
        WHERE t.team_id = ?
        GROUP BY t.id
    )
SELECT
    (SELECT JSON_OBJECTAGG(status, cnt) FROM status_counts) AS statuses_json,
    (SELECT JSON_ARRAYAGG(JSON_OBJECT('assignee_id', assignee_id, 'closed_count', closed_count))
       FROM top_assignees)                                  AS top_assignees_json,
    (SELECT avg_sec FROM avg_close)                         AS avg_close_sec,
    (SELECT JSON_OBJECTAGG(task_id, cnt) FROM comment_counts) AS comments_json
`

	var (
		statusesJSON sql.NullString
		topJSON      sql.NullString
		avgCloseSec  sql.NullFloat64
		commentsJSON sql.NullString
	)

	err := r.db.QueryRowxContext(ctx, q, teamID, teamID, teamID, teamID).Scan(
		&statusesJSON, &topJSON, &avgCloseSec, &commentsJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("query team stats: %w", err)
	}

	resp := &model.TeamStatsResponse{
		TeamID:         teamID,
		TasksByStatus:  map[string]int{},
		TopAssignees:   []model.TopAssignee{},
		CommentsByTask: map[string]int{},
	}

	// JSON_OBJECTAGG возвращает NULL, если строк нет → пустая map.
	if statusesJSON.Valid && statusesJSON.String != "" {
		_ = json.Unmarshal([]byte(statusesJSON.String), &resp.TasksByStatus)
	}
	if topJSON.Valid && topJSON.String != "" {
		_ = json.Unmarshal([]byte(topJSON.String), &resp.TopAssignees)
	}
	if commentsJSON.Valid && commentsJSON.String != "" {
		_ = json.Unmarshal([]byte(commentsJSON.String), &resp.CommentsByTask)
	}
	if avgCloseSec.Valid {
		v := avgCloseSec.Float64
		resp.AvgCloseTimeSec = &v
	}

	return resp, nil
}
