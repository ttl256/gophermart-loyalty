package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ttl256/gophermart-loyalty/internal/auth"
	"github.com/ttl256/gophermart-loyalty/internal/domain"
)

type UserRepo interface {
	CreateUser(ctx context.Context, user domain.User, password auth.PasswordHash) (uuid.UUID, error)
	GetUserByLogin(ctx context.Context, login string) (domain.User, auth.PasswordHash, error)
}

type AuthService struct {
	repo UserRepo
}

func NewAuthService(repo UserRepo) *AuthService {
	return &AuthService{
		repo: repo,
	}
}

func (s *AuthService) RegisterUser(ctx context.Context, user domain.User, password string) (uuid.UUID, error) {
	hash, err := auth.NewHashPassword(password)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("hashing password: %w", err)
	}
	id, err := s.repo.CreateUser(ctx, user, hash)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("register user: %w", err)
	}
	return id, nil
}

func (s *AuthService) LoginUser(ctx context.Context, login string, password string) (domain.User, error) {
	user, hash, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return domain.User{}, fmt.Errorf("getting user: %w", err)
	}
	ok, err := hash.ComparePassword(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("comparing hash: %w", err)
	}
	if !ok {
		return domain.User{}, domain.ErrInvalidCredentials
	}
	return user, nil
}
