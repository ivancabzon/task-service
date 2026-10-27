package model

import (
	"encoding/json"
	"time"
)

// ============================================================
//
//	КОНСТАНТЫ РОЛЕЙ
//
// ============================================================
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// ============================================================
//
//	КОНСТАНТЫ СТАТУСОВ ЗАДАЧ
//
// ============================================================
const (
	StatusTodo       = "todo"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
	StatusClosed     = "closed"
)

// AllStatuses — для валидации.
var AllStatuses = map[string]bool{
	StatusTodo:       true,
	StatusInProgress: true,
	StatusDone:       true,
	StatusClosed:     true,
}

// ============================================================
//  БД-МОДЕЛИ
// ============================================================

type User struct {
	ID           int64     `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"` // НИКОГДА не отдаём наружу
	Name         string    `db:"name" json:"name"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type Team struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedBy int64     `db:"created_by" json:"created_by"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type TeamMember struct {
	TeamID    int64     `db:"team_id" json:"team_id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Role      string    `db:"role" json:"role"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Task struct {
	ID          int64      `db:"id" json:"id"`
	TeamID      int64      `db:"team_id" json:"team_id"`
	Title       string     `db:"title" json:"title"`
	Description *string    `db:"description" json:"description"`
	Status      string     `db:"status" json:"status"`
	CreatedBy   int64      `db:"created_by" json:"created_by"`
	AssigneeID  *int64     `db:"assignee_id" json:"assignee_id"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
	ClosedAt    *time.Time `db:"closed_at" json:"closed_at"`
	Version     int        `db:"version" json:"version"`
}

type TaskHistory struct {
	ID        int64           `db:"id" json:"id"`
	TaskID    int64           `db:"task_id" json:"task_id"`
	ChangedBy int64           `db:"changed_by" json:"changed_by"`
	Changes   json.RawMessage `db:"changes" json:"changes" swaggertype:"object"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

type FieldChange struct {
	Old any `json:"old"`
	New any `json:"new"`
}

type ChangesMap map[string]FieldChange

type TaskComment struct {
	ID        int64     `db:"id" json:"id"`
	TaskID    int64     `db:"task_id" json:"task_id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Content   string    `db:"content" json:"content"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// ============================================================
//  DTO — запросы и ответы
// ============================================================

type RegisterRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=72"`
	Name     string `json:"name"     binding:"required,min=1,max=255"`
}

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type CreateTeamRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
}

type InviteRequest struct {
	UserID int64  `json:"user_id" binding:"required,gt=0"`
	Role   string `json:"role"    binding:"required,oneof=admin member"`
}

type CreateTaskRequest struct {
	TeamID      int64   `json:"team_id"     binding:"required,gt=0"`
	Title       string  `json:"title"       binding:"required,min=1,max=255"`
	Description *string `json:"description" binding:"omitempty,max=10000"`
	AssigneeID  *int64  `json:"assignee_id"`
}

// UpdateTaskRequest — все поля опциональны (nil = не менять).
//   - AssigneeID: nil = не менять, 0 = очистить, >0 = назначить
//   - Version обязателен для оптимистичной блокировки.
type UpdateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"      binding:"omitempty,oneof=todo in_progress done closed"`
	AssigneeID  *int64  `json:"assignee_id"`
	Version     int     `json:"version"     binding:"required,gt=0"`
}

type TaskFilter struct {
	TeamID     int64
	Status     string
	AssigneeID *int64
	Limit      int
	Offset     int
}

type AddCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=10000"`
}

// TeamStatsResponse — ответ GET /teams/:id/stats.
type TeamStatsResponse struct {
	TeamID          int64          `json:"team_id"`
	TasksByStatus   map[string]int `json:"tasks_by_status"`
	TopAssignees    []TopAssignee  `json:"top_assignees_30d"`
	AvgCloseTimeSec *float64       `json:"avg_close_time_seconds"`
	CommentsByTask  map[string]int `json:"comments_by_task"`
}

type TopAssignee struct {
	AssigneeID  int64 `json:"assignee_id"  db:"assignee_id"`
	ClosedCount int   `json:"closed_count" db:"closed_count"`
}
