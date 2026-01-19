package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type User struct {
	ID    uuid.UUID
	Login string
}

func NewUser(login string) User {
	return User{
		ID:    uuid.New(),
		Login: login,
	}
}

type Order struct {
	Number     OrderNumber
	Status     OrderStatus
	UserID     uuid.UUID
	Accrual    decimal.Decimal
	UploadedAt time.Time
}

// OrderStatus ENUM(NEW, PROCESSING, INVALID, PROCESSED).
type OrderStatus int //nolint: recvcheck //fine

type OrderNumber string

func NewOrderNumber(s string) (OrderNumber, error) {
	if !ValidLuhn(s) {
		return "", ErrMalformedOrderNumber
	}
	return OrderNumber(s), nil
}

//nolint:mnd //fine
func ValidLuhn(s string) bool {
	sum := 0
	double := false
	digits := 0

	for i := len(s) - 1; i >= 0; i-- {
		c := s[i]

		if c == ' ' || c == '-' {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}

		n := int(c - '0')
		digits++

		if double {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		double = !double
	}

	return sum%10 == 0
}
