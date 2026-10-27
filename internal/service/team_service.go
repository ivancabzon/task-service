package service

import (
	"context"
	"strings"

	"github.com/ivancabzon/task-service/internal/model"
	"github.com/ivancabzon/task-service/pkg/apperr"
)

type TeamRepository interface {
	CreateWithOwner(ctx context.Context, t *model.Team) (int64, error)
	GetByID(ctx context.Context, id int64) (*model.Team, error)
	GetUserRole(ctx context.Context, teamID, userID int64) (string, error)
	ListByUser(ctx context.Context, userID int64) ([]model.Team, error)
	AddMember(ctx context.Context, teamID, userID int64, role string) error
}

type TeamService struct {
	teams TeamRepository
	users UserRepository
}

func NewTeamService(teams TeamRepository, users UserRepository) *TeamService {
	return &TeamService{teams: teams, users: users}
}

func (s *TeamService) Create(ctx context.Context, userID int64, req model.CreateTeamRequest) (*model.Team, error) {
	t := &model.Team{
		Name:      strings.TrimSpace(req.Name),
		CreatedBy: userID,
	}

	id, err := s.teams.CreateWithOwner(ctx, t)
	if err != nil {
		return nil, err
	}

	created, err := s.teams.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *TeamService) ListMine(ctx context.Context, userID int64) ([]model.Team, error) {
	return s.teams.ListByUser(ctx, userID)
}

// Только owner или admin. Роль owner через invite запрещена.
func (s *TeamService) Invite(ctx context.Context, inviterID, teamID int64, req model.InviteRequest) error {
	// 1. Проверяем роль приглашающего
	role, err := s.teams.GetUserRole(ctx, teamID, inviterID)
	if err != nil {
		return err // Forbidden, если не в команде
	}
	if role != model.RoleOwner && role != model.RoleAdmin {
		return apperr.Forbidden("only owner or admin can invite")
	}

	if req.Role == model.RoleOwner {
		return apperr.Validation("cannot invite as owner")
	}

	if _, err := s.users.FindByID(ctx, req.UserID); err != nil {
		return apperr.NotFound("target user not found")
	}

	return s.teams.AddMember(ctx, teamID, req.UserID, req.Role)
}

func (s *TeamService) EnsureRole(ctx context.Context, teamID, userID int64, allowed ...string) (string, error) {
	role, err := s.teams.GetUserRole(ctx, teamID, userID)
	if err != nil {
		return "", err
	}
	for _, a := range allowed {
		if role == a {
			return role, nil
		}
	}
	return "", apperr.Forbidden("insufficient role for this action")
}
