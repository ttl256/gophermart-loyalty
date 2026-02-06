package config_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/alexflint/go-arg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ttl256/gophermart-loyalty/internal/config"
)

func TestParseConfig(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		want := config.NewServer()
		got, err := config.BuildConfig(nil, nil)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("use env", func(t *testing.T) {
		want := config.NewServer()
		want.Address = "localhost:80"
		want.DSN = "test_uri"
		want.AccrualAddress = "http://localhost:8082"
		want.LogLevel = slog.LevelDebug

		t.Setenv("RUN_ADDRESS", want.Address)
		t.Setenv("DATABASE_URI", want.DSN)
		t.Setenv("ACCRUAL_SYSTEM_ADDRESS", want.AccrualAddress)
		t.Setenv("LOG_LEVEL", want.LogLevel.String())

		got, err := config.BuildConfig(nil, nil)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("use flags", func(t *testing.T) {
		want := config.NewServer()
		want.Address = "localhost:80"
		want.DSN = "test_uri"
		want.AccrualAddress = "http://localhost:8082"
		want.LogLevel = slog.LevelDebug

		got, err := config.BuildConfig(
			[]string{
				"-a", want.Address,
				"-d", want.DSN,
				"-r", want.AccrualAddress,
				"--loglevel", want.LogLevel.String(),
			}, nil,
		)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("flags precede env", func(t *testing.T) {
		want := config.NewServer()
		want.Address = "localhost:80"
		want.DSN = "test_uri"
		want.AccrualAddress = "http://localhost:8082"
		want.LogLevel = slog.LevelDebug

		t.Setenv("RUN_ADDRESS", "garbage")
		t.Setenv("DATABASE_URI", "garbage")
		t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "garbage")
		t.Setenv("LOG_LEVEL", "info")

		got, err := config.BuildConfig(
			[]string{
				"-a", want.Address,
				"-d", want.DSN,
				"-r", want.AccrualAddress,
				"--loglevel", want.LogLevel.String(),
			}, nil,
		)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("get some help", func(t *testing.T) {
		var buf bytes.Buffer
		_, err := config.BuildConfig([]string{"-h"}, &buf)
		require.ErrorIs(t, err, arg.ErrHelp)
		bufS := buf.String()
		assert.Contains(t, bufS, "Usage:")
		assert.Contains(t, bufS, "Options:")
		assert.Contains(t, bufS, "--help")
	})
}
