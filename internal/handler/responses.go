package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"github.com/ttl256/gophermart-loyalty/internal/domain"
)

// HealthStatus ENUM(ok).
type HealthStatus int //nolint: recvcheck //fine

type HealthResponse struct {
	Status HealthStatus `json:"status"`
}

type Money decimal.Decimal //nolint: recvcheck //json

func (m Money) MarshalJSON() ([]byte, error) {
	s := decimal.Decimal(m).String()
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing money: %s: %w", s, err)
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encoding money: %v: %w", v, err)
	}
	return data, nil
}

func (m *Money) UnmarshalJSON(b []byte) error {
	var v float64
	err := json.Unmarshal(b, &v)
	if err != nil {
		return fmt.Errorf("decoding money: %w", err)
	}
	*m = Money(decimal.NewFromFloat(v))
	return nil
}

func (m Money) IsZero() bool {
	return decimal.Decimal(m).IsZero()
}

type OrderResponse struct {
	Number     domain.OrderNumber `json:"number"`
	Status     domain.OrderStatus `json:"status"`
	Accrual    Money              `json:"accrual,omitzero"`
	UploadedAt time.Time          `json:"uploaded_at"`
}

type BalanceResponse struct {
	Current   Money `json:"current"`
	Withdrawn Money `json:"withdrawn"`
}
