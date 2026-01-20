package handler

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/ttl256/gophermart-loyalty/internal/domain"
)

// HealthStatus ENUM(ok).
type HealthStatus int //nolint: recvcheck //fine

type HealthResponse struct {
	Status HealthStatus `json:"status"`
}

type OrderResponse struct {
	Number     domain.OrderNumber `json:"number"`
	Status     domain.OrderStatus `json:"status"`
	Accrual    decimal.Decimal    `json:"accrual,omitzero"`
	UploadedAt time.Time          `json:"uploaded_at"`
}
