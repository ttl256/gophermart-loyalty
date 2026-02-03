package accrual

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	xerrors "github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/ttl256/gophermart-loyalty/internal/domain"
)

type Repo interface {
	GetOrdersForProcessing(ctx context.Context) ([]domain.Order, error)
	UpdateOrderStatus(
		ctx context.Context,
		number domain.OrderNumber,
		status domain.OrderStatus,
		accrual decimal.Decimal,
	) error
}

type Worker struct {
	repo   Repo
	client *Client
	freq   time.Duration
	logger *slog.Logger
}

func NewWorker(repo Repo, client *Client, freq time.Duration) *Worker {
	return &Worker{
		repo:   repo,
		client: client,
		freq:   freq,
		logger: slog.Default(),
	}
}

func (w *Worker) Run(ctx context.Context) error { //nolint: gocognit //fine
	sleep := w.freq
	for {
		select {
		case <-ctx.Done():
			return xerrors.WithStack(ctx.Err())
		default:
			select {
			case <-ctx.Done():
				return xerrors.WithStack(ctx.Err())
			case <-time.After(sleep):
				sleep = w.freq
				orders, err := w.repo.GetOrdersForProcessing(ctx)
				if err != nil {
					w.logger.ErrorContext(ctx, "fetching orders for processing", slog.Any("error", err))
					return xerrors.WithStack(err)
				}
				if len(orders) == 0 {
					w.logger.DebugContext(ctx, "no orders to process")
					continue
				}
				for _, order := range orders {
					if err = w.Process(ctx, order.Number); err != nil {
						var errRateLimit RateLimitError
						if errors.As(err, &errRateLimit) {
							w.logger.InfoContext(
								ctx, "rate limit", slog.Duration("retry_after", errRateLimit.RetryAfter),
							)
							sleep = errRateLimit.RetryAfter
							break
						}
						w.logger.ErrorContext(
							ctx,
							"processing order",
							slog.String("order", string(order.Number)),
							slog.Any("error", fmt.Sprintf("%+v", err)),
						)
					}
				}
			}
		}
	}
}

func (w *Worker) Process(ctx context.Context, order domain.OrderNumber) error {
	info, found, err := w.client.GetOrder(ctx, order)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	var (
		newStatus domain.OrderStatus
		accrual   decimal.Decimal
	)

	switch info.Status {
	case OrderStatusPROCESSED:
		newStatus = domain.OrderStatusPROCESSED
		accrual = decimal.Decimal(info.Accrual)
	case OrderStatusPROCESSING, OrderStatusREGISTERED:
		newStatus = domain.OrderStatusPROCESSING
		accrual = decimal.Zero
	case OrderStatusINVALID:
		newStatus = domain.OrderStatusINVALID
		accrual = decimal.Zero
	default:
		return fmt.Errorf("unexpected accrual status %q for order %q", info.Status.String(), string(order))
	}

	if err = w.repo.UpdateOrderStatus(ctx, order, newStatus, accrual); err != nil {
		return fmt.Errorf("updating order %q status: %w", string(order), err)
	}
	return nil
}
