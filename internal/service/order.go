package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ttl256/gophermart-loyalty/internal/domain"
)

type OrderRepo interface {
	RegisterOrder(ctx context.Context, userID uuid.UUID, order domain.OrderNumber) (uuid.UUID, error)
	GetOrders(ctx context.Context, userID uuid.UUID) ([]domain.Order, error)
}

type OrderService struct {
	repo OrderRepo
}

func NewOrderService(repo OrderRepo) *OrderService {
	return &OrderService{
		repo: repo,
	}
}

func (s *OrderService) RegisterOrder(
	ctx context.Context,
	userID uuid.UUID,
	order domain.OrderNumber,
) (uuid.UUID, error) {
	id, err := s.repo.RegisterOrder(ctx, userID, order)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("register order: %w", err)
	}
	return id, nil
}

func (s *OrderService) GetOrders(ctx context.Context, userID uuid.UUID) ([]domain.Order, error) {
	orders, err := s.repo.GetOrders(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting orders: %w", err)
	}
	return orders, nil
}
