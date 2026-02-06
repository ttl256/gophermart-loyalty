package domain_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ttl256/gophermart-loyalty/internal/domain"
)

func TestNewOrder(t *testing.T) {
	cases := []struct {
		input string
		want  error
	}{
		{"49927398716", nil},
		{"49927398717", domain.ErrMalformedOrderNumber},
		{"49a92b73c98d717e", domain.ErrMalformedOrderNumber},
	}

	for i, tt := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := domain.NewOrderNumber(tt.input)
			require.ErrorIs(t, err, tt.want)
		})
	}
}
