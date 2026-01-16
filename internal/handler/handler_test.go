package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ttl256/gophermart-loyalty/internal/handler"
	"resty.dev/v3"
)

func TestHealthHandler(t *testing.T) {
	t.Parallel()

	h := handler.NewHTTPHandler()
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(func() {
		srv.Close()
	})

	want := handler.HealthResponse{Status: handler.HealthStatusOk}

	client := resty.New().SetBaseURL(srv.URL)
	var got handler.HealthResponse
	resp, err := client.R().SetResult(&got).Get("/healthz")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, want, got)
}
