package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ivancabzon/task-service/internal/model"
	"github.com/ivancabzon/task-service/pkg/apperr"
	"github.com/ivancabzon/task-service/pkg/jwt"
)

type UserRepository interface {
	Create(ctx context.Context, u *model.User) (int64, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id int64) (*model.User, error)
}

type AuthService struct {
	users     UserRepository
	jwtSecret string
	jwtTTL    time.Duration
}

func NewAuthService(users UserRepository, jwtSecret string, jwtTTL time.Duration) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret, jwtTTL: jwtTTL}
}

func (s *AuthService) Register(ctx context.Context, req model.RegisterRequest) (*model.User, error) {
	if len(req.Password) > 72 {
		return nil, apperr.Validation("password too long (max 72 bytes)")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u := &model.User{
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash: string(hash),
		Name:         strings.TrimSpace(req.Name),
	}

	id, err := s.users.Create(ctx, u)
	if err != nil {
		return nil, err
	}

	created, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("reload user after create: %w", err)
	}
	created.PasswordHash = ""
	return created, nil
}

func (s *AuthService) Login(ctx context.Context, req model.LoginRequest) (*model.LoginResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		// Не раскрываем, что именно неверно (email или пароль)
		return nil, apperr.Unauthorized("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperr.Unauthorized("invalid email or password")
	}

	token, err := jwt.Generate(u.ID, u.Email, s.jwtSecret, s.jwtTTL)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Скрываем хеш в ответе
	u.PasswordHash = ""

	return &model.LoginResponse{Token: token, User: *u}, nil
}
