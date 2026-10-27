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

type TeamHandler struct {
	teams *service.TeamService
}

func NewTeamHandler(teams *service.TeamService) *TeamHandler {
	return &TeamHandler{teams: teams}
}

// Create — POST /api/v1/teams
//
//	@Summary      Создать команду
//	@Tags         teams
//	@Security     BearerAuth
//	@Accept       json
//	@Produce      json
//	@Param        request body model.CreateTeamRequest true "Название"
//	@Success      201 {object} model.Team
//	@Failure      400 {object} response.ErrorBody
//	@Failure      401 {object} response.ErrorBody
//	@Router       /teams [post]
func (h *TeamHandler) Create(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "unauthorized", "no user in context")
		return
	}

	var req model.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	team, err := h.teams.Create(c.Request.Context(), uid, req)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, team)
}

// ListMine — GET /api/v1/teams
//
//	@Summary      Мои команды
//	@Tags         teams
//	@Security     BearerAuth
//	@Produce      json
//	@Success      200 {object} map[string][]model.Team
//	@Router       /teams [get]
func (h *TeamHandler) ListMine(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "unauthorized", "no user in context")
		return
	}

	teams, err := h.teams.ListMine(c.Request.Context(), uid)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, gin.H{"teams": teams})
}

// Invite — POST /api/v1/teams/:id/invite
//
//	@Summary      Пригласить в команду
//	@Tags         teams
//	@Security     BearerAuth
//	@Accept       json
//	@Produce      json
//	@Param        id path int true "Team ID"
//	@Param        request body model.InviteRequest true "Кого и кем"
//	@Success      204
//	@Failure      400 {object} response.ErrorBody
//	@Failure      403 {object} response.ErrorBody
//	@Failure      409 {object} response.ErrorBody
//	@Router       /teams/{id}/invite [post]
func (h *TeamHandler) Invite(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "unauthorized", "no user in context")
		return
	}

	teamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || teamID <= 0 {
		response.Fail(c, http.StatusBadRequest, "validation_error", "invalid team id")
		return
	}

	var req model.InviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	if err := h.teams.Invite(c.Request.Context(), uid, teamID, req); err != nil {
		response.HandleError(c, err)
		return
	}
	response.NoContent(c)
}
