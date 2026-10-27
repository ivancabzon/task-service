package service

import (
	"context"
	"strings"
	"time"

	"github.com/ivancabzon/task-service/internal/model"
	"github.com/ivancabzon/task-service/pkg/apperr"
)

type TaskRepository interface {
	Create(ctx context.Context, t *model.Task) (int64, error)
	GetByID(ctx context.Context, id int64) (*model.Task, error)
	List(ctx context.Context, f model.TaskFilter) ([]model.Task, error)
	UpdateWithHistory(ctx context.Context, t *model.Task, changes model.ChangesMap, userID int64) error
}

type HistoryRepository interface {
	ListByTask(ctx context.Context, taskID int64) ([]model.TaskHistory, error)
}

type CommentRepository interface {
	Create(ctx context.Context, c *model.TaskComment) (int64, error)
	GetByID(ctx context.Context, id int64) (*model.TaskComment, error)
	ListByTask(ctx context.Context, taskID int64) ([]model.TaskComment, error)
}

// CacheRepository — интерфейс для кеша (легко мокать в тестах).
type CacheRepository interface {
	GetTasks(ctx context.Context, f model.TaskFilter) ([]model.Task, bool, error)
	SetTasks(ctx context.Context, f model.TaskFilter, tasks []model.Task) error
	InvalidateTeamTasks(ctx context.Context, teamID int64) error
}

type TaskService struct {
	tasks    TaskRepository
	teams    TeamRepository
	cache    CacheRepository
	history  HistoryRepository
	comments CommentRepository
}

func NewTaskService(
	tasks TaskRepository,
	teams TeamRepository,
	cache CacheRepository,
	history HistoryRepository,
	comments CommentRepository,
) *TaskService {
	return &TaskService{
		tasks:    tasks,
		teams:    teams,
		cache:    cache,
		history:  history,
		comments: comments,
	}
}

// ============================================================
//  CREATE
// ============================================================

func (s *TaskService) Create(ctx context.Context, userID int64, req model.CreateTaskRequest) (*model.Task, error) {
	if _, err := s.teams.GetUserRole(ctx, req.TeamID, userID); err != nil {
		return nil, err
	}
	if req.AssigneeID != nil {
		if _, err := s.teams.GetUserRole(ctx, req.TeamID, *req.AssigneeID); err != nil {
			return nil, apperr.Validation("assignee must be a member of the team")
		}
	}

	t := &model.Task{
		TeamID:      req.TeamID,
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		Status:      model.StatusTodo,
		CreatedBy:   userID,
		AssigneeID:  req.AssigneeID,
	}
	if t.Title == "" {
		return nil, apperr.Validation("title cannot be empty")
	}

	id, err := s.tasks.Create(ctx, t)
	if err != nil {
		return nil, err
	}

	created, err := s.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// ⚡ Инвалидация ПОСЛЕ успешной записи в БД.
	// Ошибку кеша логируем, но не возвращаем клиенту —
	// запись уже произошла, кеш протухнет через TTL.
	_ = s.cache.InvalidateTeamTasks(ctx, req.TeamID)

	return created, nil
}

// ============================================================
//  LIST — cache-aside
// ============================================================

func (s *TaskService) List(ctx context.Context, userID int64, f model.TaskFilter) ([]model.Task, error) {
	if _, err := s.teams.GetUserRole(ctx, f.TeamID, userID); err != nil {
		return nil, err
	}

	// Нормализация ДО построения ключа — критично для стабильности кеша!
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if f.Status != "" && !model.AllStatuses[f.Status] {
		return nil, apperr.Validation("invalid status filter")
	}

	// 1. Пробуем кеш
	if cached, ok, err := s.cache.GetTasks(ctx, f); err == nil && ok {
		return cached, nil
	}
	// ошибки кеша игнорируем — идём в БД (fallback)

	// 2. Cache miss → БД
	tasks, err := s.tasks.List(ctx, f)
	if err != nil {
		return nil, err
	}

	// 3. Кладём в кеш (ошибку игнорируем)
	_ = s.cache.SetTasks(ctx, f, tasks)

	return tasks, nil
}

// ============================================================
//  UPDATE
// ============================================================

func (s *TaskService) Update(ctx context.Context, userID, taskID int64, req model.UpdateTaskRequest) (*model.Task, error) {
	old, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// ⚡ Optimistic lock: version из запроса ДОЛЖНА совпадать с БД
	if req.Version != old.Version {
		return nil, apperr.Conflict("task was modified by another user; reload and retry")
	}

	role, err := s.teams.GetUserRole(ctx, old.TeamID, userID)
	if err != nil {
		return nil, err
	}

	isPrivileged := role == model.RoleOwner || role == model.RoleAdmin
	isCreator := old.CreatedBy == userID
	isAssignee := old.AssigneeID != nil && *old.AssigneeID == userID

	if !isPrivileged && !isCreator && !isAssignee {
		return nil, apperr.Forbidden("you cannot edit this task")
	}

	allowTitle := isPrivileged || isCreator
	allowDesc := isPrivileged || isCreator
	allowStatus := isPrivileged || isCreator || isAssignee
	allowAssignee := isPrivileged || isCreator

	if req.Title != nil && !allowTitle {
		return nil, apperr.Forbidden("you cannot change title")
	}
	if req.Description != nil && !allowDesc {
		return nil, apperr.Forbidden("you cannot change description")
	}
	if req.Status != nil && !allowStatus {
		return nil, apperr.Forbidden("you cannot change status")
	}
	if req.AssigneeID != nil && !allowAssignee {
		return nil, apperr.Forbidden("assignee cannot reassign task")
	}

	updated := *old
	changes := model.ChangesMap{}

	if req.Title != nil {
		newTitle := strings.TrimSpace(*req.Title)
		if newTitle == "" {
			return nil, apperr.Validation("title cannot be empty")
		}
		if newTitle != old.Title {
			changes["title"] = model.FieldChange{Old: old.Title, New: newTitle}
			updated.Title = newTitle
		}
	}

	if req.Description != nil {
		var oldVal any
		if old.Description != nil {
			oldVal = *old.Description
		}
		if !strPtrEq(old.Description, req.Description) {
			changes["description"] = model.FieldChange{Old: oldVal, New: *req.Description}
			updated.Description = req.Description
		}
	}

	if req.Status != nil && *req.Status != old.Status {
		changes["status"] = model.FieldChange{Old: old.Status, New: *req.Status}
		updated.Status = *req.Status

		now := time.Now().UTC()
		if *req.Status == model.StatusClosed && old.Status != model.StatusClosed {
			updated.ClosedAt = &now
		} else if *req.Status != model.StatusClosed {
			updated.ClosedAt = nil
		}
	}

	if req.AssigneeID != nil {
		var newAssignee *int64
		if *req.AssigneeID > 0 {
			v := *req.AssigneeID
			newAssignee = &v
		}
		if !int64PtrEq(old.AssigneeID, newAssignee) {
			if newAssignee != nil {
				if _, err := s.teams.GetUserRole(ctx, old.TeamID, *newAssignee); err != nil {
					return nil, apperr.Validation("new assignee must be a member of the team")
				}
			}
			var oldVal any
			if old.AssigneeID != nil {
				oldVal = *old.AssigneeID
			}
			var newVal any
			if newAssignee != nil {
				newVal = *newAssignee
			}
			changes["assignee_id"] = model.FieldChange{Old: oldVal, New: newVal}
			updated.AssigneeID = newAssignee
		}
	}

	if len(changes) == 0 {
		return old, nil
	}

	updated.Version = req.Version
	if err := s.tasks.UpdateWithHistory(ctx, &updated, changes, userID); err != nil {
		return nil, err
	}

	// ⚡ Инвалидация после успешного commit
	_ = s.cache.InvalidateTeamTasks(ctx, old.TeamID)

	return s.tasks.GetByID(ctx, taskID)
}

// ============================================================
//  HISTORY
// ============================================================

func (s *TaskService) History(ctx context.Context, userID, taskID int64) ([]model.TaskHistory, error) {
	t, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	// Проверка членства — иначе утечка (ТЗ)
	if _, err := s.teams.GetUserRole(ctx, t.TeamID, userID); err != nil {
		return nil, err
	}
	return s.history.ListByTask(ctx, taskID)
}

// ============================================================
//  COMMENTS
// ============================================================

func (s *TaskService) AddComment(ctx context.Context, userID, taskID int64, req model.AddCommentRequest) (*model.TaskComment, error) {
	t, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	// Комментирование — это НЕ редактирование. Достаточно быть членом команды.
	if _, err := s.teams.GetUserRole(ctx, t.TeamID, userID); err != nil {
		return nil, err
	}

	c := &model.TaskComment{
		TaskID:  taskID,
		UserID:  userID,
		Content: strings.TrimSpace(req.Content),
	}
	if c.Content == "" {
		return nil, apperr.Validation("comment content cannot be empty")
	}

	id, err := s.comments.Create(ctx, c)
	if err != nil {
		return nil, err
	}
	return s.comments.GetByID(ctx, id)
}

func (s *TaskService) ListComments(ctx context.Context, userID, taskID int64) ([]model.TaskComment, error) {
	t, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if _, err := s.teams.GetUserRole(ctx, t.TeamID, userID); err != nil {
		return nil, err
	}
	return s.comments.ListByTask(ctx, taskID)
}

// ---------- helpers ----------

func strPtrEq(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func int64PtrEq(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
