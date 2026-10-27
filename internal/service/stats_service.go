package service

import (
	"context"

	"github.com/ivancabzon/task-service/internal/model"
	"github.com/ivancabzon/task-service/pkg/apperr"
)

type StatsRepository interface {
	GetTeamStats(ctx context.Context, teamID int64) (*model.TeamStatsResponse, error)
}

type StatsService struct {
	stats StatsRepository
	teams TeamRepository
}

func NewStatsService(stats StatsRepository, teams TeamRepository) *StatsService {
	return &StatsService{stats: stats, teams: teams}
}

// GetTeamStats — доступно только owner или admin команды.
func (s *StatsService) GetTeamStats(ctx context.Context, userID, teamID int64) (*model.TeamStatsResponse, error) {
	role, err := s.teams.GetUserRole(ctx, teamID, userID)
	if err != nil {
		// не член → Forbidden (не палим существование команды)
		return nil, apperr.Forbidden("access denied to this team's stats")
	}
	if role != model.RoleOwner && role != model.RoleAdmin {
		return nil, apperr.Forbidden("only owner or admin can view stats")
	}

	return s.stats.GetTeamStats(ctx, teamID)
}
