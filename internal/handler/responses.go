package handler

import (
	"time"

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
	Accrual    float64            `json:"accrual,omitzero"`
	UploadedAt time.Time          `json:"uploaded_at"`
}

type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}
