//go:build integration

package repository_test

import (
	"context"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ivancabzon/task-service/internal/repository"
)

func TestStatsRepo_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN is not set, skipping integration test")
	}

	db, err := sqlx.Connect("mysql", dsn)
	require.NoError(t, err)
	defer db.Close()

	// 1. Накатываем схему (файл лежит в корне проекта)
	schema, err := os.ReadFile("../../migrations/001_init.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(schema))
	require.NoError(t, err, "failed to apply schema")

	// 2. Очищаем БД перед тестом (порядок важен из-за FK)
	_, err = db.Exec(`
		SET FOREIGN_KEY_CHECKS = 0;
		TRUNCATE TABLE task_history;
		TRUNCATE TABLE task_comments;
		TRUNCATE TABLE tasks;
		TRUNCATE TABLE team_members;
		TRUNCATE TABLE teams;
		TRUNCATE TABLE users;
		SET FOREIGN_KEY_CHECKS = 1;
	`)
	require.NoError(t, err, "failed to clean DB")

	// 3. Seed-данные
	seedIntegrationTest(t, db)

	// 4. Гоняем отчёт
	repo := repository.NewStatsRepo(db)
	stats, err := repo.GetTeamStats(context.Background(), 1)
	require.NoError(t, err)

	// 5. Ассерты
	assert.Equal(t, int64(1), stats.TeamID)

	assert.Equal(t, map[string]int{
		"todo":        1,
		"in_progress": 1,
		"done":        1,
		"closed":      4,
	}, stats.TasksByStatus)

	require.Len(t, stats.TopAssignees, 2)
	assert.Equal(t, int64(3), stats.TopAssignees[0].AssigneeID)
	assert.Equal(t, 2, stats.TopAssignees[0].ClosedCount)
	assert.Equal(t, int64(1), stats.TopAssignees[1].AssigneeID)
	assert.Equal(t, 1, stats.TopAssignees[1].ClosedCount)

	require.NotNil(t, stats.AvgCloseTimeSec)
	assert.InDelta(t, 626400.0, *stats.AvgCloseTimeSec, 1.0, "avg close time mismatch")

	nonZero := 0
	totalComments := 0
	for _, v := range stats.CommentsByTask {
		if v > 0 {
			nonZero++
			totalComments += v
		}
	}
	assert.Equal(t, 2, nonZero, "exactly 2 tasks with comments")
	assert.Equal(t, 3, totalComments, "1 + 2 = 3 comments total")

	// Проверка изоляции: команда 2 видит только свои задачи
	stats2, err := repo.GetTeamStats(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"closed": 1}, stats2.TasksByStatus)
}

func seedIntegrationTest(t *testing.T, db *sqlx.DB) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, name) VALUES
		(1, 'alice@test.io', 'x', 'Alice'),
		(2, 'admin@test.io', 'x', 'Admin'),
		(3, 'bob@test.io',   'x', 'Bob');

		INSERT INTO teams (id, name, created_by) VALUES
		(1, 'Team One', 1),
		(2, 'Team Two', 1);

		INSERT INTO team_members (team_id, user_id, role) VALUES
		(1, 1, 'owner'),
		(1, 3, 'member'),
		(2, 1, 'owner');
	`)
	require.NoError(t, err)

	// Явно задаем ID, чтобы комментарии встали куда надо
	_, err = db.Exec(`
		INSERT INTO tasks
			(id, team_id, title, status, created_by, assignee_id, created_at, closed_at, version)
		VALUES
			(1, 1, 'T1 todo',         'todo',        1, NULL, NOW() - INTERVAL 10 DAY, NULL, 1),
			(2, 1, 'T2 in_progress',  'in_progress', 1,    3, NOW() - INTERVAL  9 DAY, NULL, 1),
			(3, 1, 'T3 done',         'done',        1,    3, NOW() - INTERVAL  8 DAY, NULL, 1),
			(4, 1, 'T4 closed bob',   'closed',      1,    3, NOW() - INTERVAL  7 DAY, NOW() - INTERVAL  5 DAY, 1),
			(5, 1, 'T5 closed bob',   'closed',      1,    3, NOW() - INTERVAL  6 DAY, NOW() - INTERVAL  3 DAY, 1),
			(6, 1, 'T6 closed alice', 'closed',      1,    1, NOW() - INTERVAL  5 DAY, NOW() - INTERVAL  1 DAY, 1),
			(7, 1, 'T7 closed old',   'closed',      1,    3, NOW() - INTERVAL 60 DAY, NOW() - INTERVAL 40 DAY, 1),
			(8, 2, 'Other team task', 'closed',      1,    1, NOW() - INTERVAL  5 DAY, NOW() - INTERVAL  1 DAY, 1);

		INSERT INTO task_comments (task_id, user_id, content) VALUES
		(4, 1, 'c1'),
		(5, 3, 'c2'),
		(5, 3, 'c3');
	`)
	require.NoError(t, err)
}
