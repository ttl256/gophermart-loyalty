package accrual_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ttl256/gophermart-loyalty/internal/accrual"
)

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()
	const defaultDuration = 1 * time.Minute
	cases := []struct {
		input string
		want  time.Duration
	}{
		{"", defaultDuration},
		{"60", defaultDuration},
		{"25", 25 * time.Second},
		{"0", 0 * time.Second},
		{"-25", defaultDuration},
		{"garbage", defaultDuration},
	}

	for _, tt := range cases {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := accrual.ParseRetryAfter(tt.input, defaultDuration)
			assert.Equal(t, tt.want, got)
		})
	}
}
