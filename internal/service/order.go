package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	xerrors "github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/ttl256/gophermart-loyalty/internal/domain"
)

type OrderRepo interface {
	RegisterOrder(ctx context.Context, userID uuid.UUID, order domain.OrderNumber) (uuid.UUID, error)
	GetOrders(ctx context.Context, userID uuid.UUID) ([]domain.Order, error)
	GetBalance(ctx context.Context, userID uuid.UUID) (domain.Balance, error)
	Withdraw(ctx context.Context, userID uuid.UUID, order domain.OrderNumber, sum decimal.Decimal) error
	GetWithdrawals(ctx context.Context, userID uuid.UUID) ([]domain.Withdrawal, error)
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

func (s *OrderService) GetBalance(ctx context.Context, userID uuid.UUID) (domain.Balance, error) {
	balance, err := s.repo.GetBalance(ctx, userID)
	if err != nil {
		return domain.Balance{}, xerrors.WithStack(err)
	}
	return balance, nil
}

func (s *OrderService) Withdraw(
	ctx context.Context,
	userID uuid.UUID,
	order domain.OrderNumber,
	sum decimal.Decimal,
) error {
	err := s.repo.Withdraw(ctx, userID, order, sum)
	if err != nil {
		return fmt.Errorf("withdrawal: %w", err)
	}
	return nil
}

func (s *OrderService) GetWithdrawals(ctx context.Context, userID uuid.UUID) ([]domain.Withdrawal, error) {
	withdrawals, err := s.repo.GetWithdrawals(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting withdrawals: %w", err)
	}
	return withdrawals, nil
}
