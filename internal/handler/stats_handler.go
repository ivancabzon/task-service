package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ivancabzon/task-service/internal/middleware"
	"github.com/ivancabzon/task-service/internal/service"
	"github.com/ivancabzon/task-service/pkg/response"
)

type StatsHandler struct {
	stats *service.StatsService
}

func NewStatsHandler(stats *service.StatsService) *StatsHandler {
	return &StatsHandler{stats: stats}
}

// GetTeamStats — GET /api/v1/teams/:team_id/stats
//
//	@Summary      SQL-отчет по команде
//	@Description  Задачи по статусам, топ-3 исполнителя за 30 дней, среднее время закрытия, комментарии по задачам.
//	@Tags         stats
//	@Security     BearerAuth
//	@Produce      json
//	@Param        team_id path int true "Team ID"
//	@Success      200 {object} model.TeamStatsResponse
//	@Failure      403 {object} response.ErrorBody
//	@Failure      404 {object} response.ErrorBody
//	@Router       /teams/{team_id}/stats [get]
func (h *StatsHandler) GetTeamStats(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "unauthorized", "no user in context")
		return
	}

	teamID, err := strconv.ParseInt(c.Param("team_id"), 10, 64)
	if err != nil || teamID <= 0 {
		response.Fail(c, http.StatusBadRequest, "validation_error", "invalid team_id")
		return
	}

	stats, err := h.stats.GetTeamStats(c.Request.Context(), uid, teamID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.OK(c, stats)
}
