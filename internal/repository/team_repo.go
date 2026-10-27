package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/ivancabzon/task-service/internal/model"
	"github.com/ivancabzon/task-service/pkg/apperr"
)

type TeamRepo struct {
	db *sqlx.DB
}

func NewTeamRepo(db *sqlx.DB) *TeamRepo {
	return &TeamRepo{db: db}
}

func (r *TeamRepo) CreateWithOwner(ctx context.Context, t *model.Team) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() // no-op после Commit

	res, err := tx.ExecContext(ctx,
		`INSERT INTO teams (name, created_by) VALUES (?, ?)`,
		t.Name, t.CreatedBy)
	if err != nil {
		return 0, fmt.Errorf("insert team: %w", err)
	}

	teamID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("team last insert id: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, 'owner')`,
		teamID, t.CreatedBy)
	if err != nil {
		return 0, fmt.Errorf("insert owner membership: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit team: %w", err)
	}
	return teamID, nil
}

func (r *TeamRepo) GetByID(ctx context.Context, id int64) (*model.Team, error) {
	var t model.Team
	err := r.db.GetContext(ctx, &t, `SELECT * FROM teams WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.NotFound("team not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	return &t, nil
}

func (r *TeamRepo) GetUserRole(ctx context.Context, teamID, userID int64) (string, error) {
	var role string
	err := r.db.GetContext(ctx, &role,
		`SELECT role FROM team_members WHERE team_id = ? AND user_id = ?`,
		teamID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", apperr.Forbidden("not a team member")
	}
	if err != nil {
		return "", fmt.Errorf("get user role: %w", err)
	}
	return role, nil
}

func (r *TeamRepo) ListByUser(ctx context.Context, userID int64) ([]model.Team, error) {
	teams := make([]model.Team, 0)
	err := r.db.SelectContext(ctx, &teams, `
		SELECT t.* FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = ?
		ORDER BY t.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user teams: %w", err)
	}
	return teams, nil
}

func (r *TeamRepo) AddMember(ctx context.Context, teamID, userID int64, role string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`,
		teamID, userID, role)
	if err != nil {
		var myErr *mysql.MySQLError
		if errors.As(err, &myErr) && myErr.Number == 1062 {
			return apperr.Conflict("user is already a member of this team")
		}
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}
