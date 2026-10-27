package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ivancabzon/task-service/internal/middleware"
	"github.com/ivancabzon/task-service/internal/model"
	"github.com/ivancabzon/task-service/internal/service"
	"github.com/ivancabzon/task-service/pkg/response"
)

type TaskHandler struct {
	tasks *service.TaskService
}

func NewTaskHandler(tasks *service.TaskService) *TaskHandler {
	return &TaskHandler{tasks: tasks}
}

// Create — POST /api/v1/tasks
//
//	@Summary      Создать задачу
//	@Tags         tasks
//	@Security     BearerAuth
//	@Accept       json
//	@Produce      json
//	@Param        request body model.CreateTaskRequest true "Задача"
//	@Success      201 {object} model.Task
//	@Failure      400 {object} response.ErrorBody
//	@Failure      403 {object} response.ErrorBody
//	@Router       /tasks [post]
func (h *TaskHandler) Create(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "unauthorized", "no user in context")
		return
	}
	var req model.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	t, err := h.tasks.Create(c.Request.Context(), uid, req)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, t)
}

// List — GET /api/v1/tasks?team_id=1&status=todo&assignee_id=5&limit=20&offset=0
//
//	@Summary      Список задач команды
//	@Tags         tasks
//	@Security     BearerAuth
//	@Produce      json
//	@Param        team_id     query int    true  "Team ID"
//	@Param        status      query string false "todo|in_progress|done|closed"
//	@Param        assignee_id query int    false "Assignee ID"
//	@Param        limit       query int    false "1..100 (default 20)"
//	@Param        offset      query int    false "default 0"
//	@Success      200 {object} map[string][]model.Task
//	@Router       /tasks [get]
func (h *TaskHandler) List(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "unauthorized", "no user in context")
		return
	}

	teamID, err := strconv.ParseInt(c.Query("team_id"), 10, 64)
	if err != nil || teamID <= 0 {
		response.Fail(c, http.StatusBadRequest, "validation_error", "team_id is required and must be positive")
		return
	}

	f := model.TaskFilter{
		TeamID: teamID,
		Status: c.Query("status"),
	}
	if v := c.Query("assignee_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			response.Fail(c, http.StatusBadRequest, "validation_error", "invalid assignee_id")
			return
		}
		f.AssigneeID = &id
	}
	if v := c.Query("limit"); v != "" {
		n, _ := strconv.Atoi(v)
		f.Limit = n
	}
	if v := c.Query("offset"); v != "" {
		n, _ := strconv.Atoi(v)
		f.Offset = n
	}

	tasks, err := h.tasks.List(c.Request.Context(), uid, f)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, gin.H{"tasks": tasks})
}

// Update — PUT /api/v1/tasks/:id
//
//	@Summary      Обновить задачу (optimistic lock по version)
//	@Tags         tasks
//	@Security     BearerAuth
//	@Accept       json
//	@Produce      json
//	@Param        id      path int                     true "Task ID"
//	@Param        request body model.UpdateTaskRequest true "Поля + version"
//	@Success      200 {object} model.Task
//	@Failure      400 {object} response.ErrorBody
//	@Failure      403 {object} response.ErrorBody
//	@Failure      409 {object} response.ErrorBody
//	@Router       /tasks/{id} [put]
func (h *TaskHandler) Update(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "unauthorized", "no user in context")
		return
	}

	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		response.Fail(c, http.StatusBadRequest, "validation_error", "invalid task id")
		return
	}

	var req model.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	t, err := h.tasks.Update(c.Request.Context(), uid, taskID, req)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, t)
}

// History — GET /api/v1/tasks/:id/history
//
//	@Summary      История изменений задачи
//	@Tags         tasks
//	@Security     BearerAuth
//	@Produce      json
//	@Param        id path int true "Task ID"
//	@Success      200 {object} map[string][]model.TaskHistory
//	@Failure      403 {object} response.ErrorBody
//	@Failure      404 {object} response.ErrorBody
//	@Router       /tasks/{id}/history [get]
func (h *TaskHandler) History(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "unauthorized", "no user in context")
		return
	}
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		response.Fail(c, http.StatusBadRequest, "validation_error", "invalid task id")
		return
	}

	rows, err := h.tasks.History(c.Request.Context(), uid, taskID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, gin.H{"history": rows})
}

// AddComment — POST /api/v1/tasks/:id/comments
//
//	@Summary      Добавить комментарий
//	@Tags         tasks
//	@Security     BearerAuth
//	@Accept       json
//	@Produce      json
//	@Param        id      path int                      true "Task ID"
//	@Param        request body model.AddCommentRequest  true "Текст комментария"
//	@Success      201 {object} model.TaskComment
//	@Failure      400 {object} response.ErrorBody
//	@Failure      403 {object} response.ErrorBody
//	@Router       /tasks/{id}/comments [post]
func (h *TaskHandler) AddComment(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "unauthorized", "no user in context")
		return
	}
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		response.Fail(c, http.StatusBadRequest, "validation_error", "invalid task id")
		return
	}

	var req model.AddCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	comment, err := h.tasks.AddComment(c.Request.Context(), uid, taskID, req)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, comment)
}

// ListComments — GET /api/v1/tasks/:id/comments
//
//	@Summary      Комментарии задачи
//	@Tags         tasks
//	@Security     BearerAuth
//	@Produce      json
//	@Param        id path int true "Task ID"
//	@Success      200 {object} map[string][]model.TaskComment
//	@Failure      403 {object} response.ErrorBody
//	@Router       /tasks/{id}/comments [get]
func (h *TaskHandler) ListComments(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "unauthorized", "no user in context")
		return
	}
	taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || taskID <= 0 {
		response.Fail(c, http.StatusBadRequest, "validation_error", "invalid task id")
		return
	}

	comments, err := h.tasks.ListComments(c.Request.Context(), uid, taskID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, gin.H{"comments": comments})
}
