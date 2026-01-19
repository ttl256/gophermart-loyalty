package handler_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ttl256/gophermart-loyalty/internal/auth"
	"github.com/ttl256/gophermart-loyalty/internal/handler"
	"github.com/ttl256/gophermart-loyalty/internal/logger"
	"github.com/ttl256/gophermart-loyalty/internal/repository"
	"github.com/ttl256/gophermart-loyalty/internal/service"
	"github.com/ttl256/gophermart-loyalty/internal/testutil"
	"resty.dev/v3"
)

func TestHealthHandler(t *testing.T) {
	t.Parallel()

	h := handler.HTTPHandler{} //nolint: exhaustruct //fine
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

func TestRegisterLoginHandler(t *testing.T) {
	ctx := context.Background()
	pg, err := testutil.StartPG(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		errClose := pg.Close(ctx)
		if errClose != nil {
			t.Logf("terminating postgres container: %v", err)
		}
	})

	_ = logger.Initialize(slog.LevelInfo)

	repo, err := repository.NewDBStorage(ctx, pg.DSN)
	require.NoError(t, err)
	t.Cleanup(repo.Close)
	require.NoError(t, repo.RepoPing(ctx))
	require.NoError(t, repo.Migrate())

	authSvc := service.NewAuthService(repo)
	authManager := auth.NewManager("test", 1*time.Hour)

	h := handler.HTTPHandler{
		AuthService: authSvc,
		JWT:         authManager,
		Logger:      slog.Default(),
	}
	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)

	client := resty.New().SetBaseURL(srv.URL)
	t.Cleanup(func() {
		errClient := client.Close()
		if errClient != nil {
			t.Logf("closing http client: %v", err)
		}
	})

	t.Run("registration and login", func(t *testing.T) {
		var (
			resp         *resty.Response
			registerUUID uuid.UUID
			loginUUID    uuid.UUID
		)
		registerReq := handler.RegisterRequest{Login: "user1", Password: "passwd1"}
		resp, err = client.R().SetBody(registerReq).Post("/api/user/register")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode())
		registerUUID, err = getAuthCookie(authManager, resp.Cookies())
		require.NoError(t, err, "unable to get Authorization cookie")

		loginReq := registerReq
		resp, err = client.R().SetBody(loginReq).Post("/api/user/login")
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode())
		loginUUID, err = getAuthCookie(authManager, resp.Cookies())
		require.NoError(t, err, "unable to get Authorization cookie")

		assert.Equal(
			t,
			registerUUID,
			loginUUID,
			"User ID expected to match for registration cookie and login cookie",
		)
	})

	t.Run("unauthorized", func(t *testing.T) {
		var resp *resty.Response
		loginReqInvalidPassword := handler.RegisterRequest{Login: "user1", Password: "passwd2"}
		resp, err = client.R().SetBody(loginReqInvalidPassword).Post("/api/user/login")
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode())

		loginReqInvalidLogin := handler.RegisterRequest{Login: "user2", Password: "passwd1"}
		resp, err = client.R().SetBody(loginReqInvalidLogin).Post("/api/user/login")
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode())
	})

	t.Run("register with existing login", func(t *testing.T) {
		var resp *resty.Response
		registerReqConflict := handler.RegisterRequest{Login: "user1", Password: "passwd2"}
		resp, err = client.R().SetBody(registerReqConflict).Post("/api/user/register")
		require.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode())
	})

	t.Run("valid JSON with empty fields", func(t *testing.T) {
		var resp *resty.Response
		emptyRequests := []handler.RegisterRequest{
			{Login: "", Password: "passwd1"},
			{Login: "user1", Password: ""},
			{Login: "", Password: ""},
		}
		for _, req := range emptyRequests {
			resp, err = client.R().SetBody(req).Post("/api/user/register")
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode())
			resp, err = client.R().SetBody(req).Post("/api/user/login")
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode())
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		type ReqIncorrectFields struct {
			Login    string `json:"loginn"`
			Password string `json:"passwd"`
		}
		type ReqPartialFields1 struct {
			Login string `json:"login"`
		}
		type ReqPartialFields2 struct {
			Password string `json:"password"`
		}
		var resp *resty.Response
		for _, uri := range []string{"/api/user/register", "/api/user/login"} {
			resp, err = client.R().SetBody(ReqIncorrectFields{Login: "login", Password: "password"}).Post(uri)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode())
			resp, err = client.R().SetBody(ReqPartialFields1{Login: "login"}).Post(uri)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode())
			resp, err = client.R().SetBody(ReqPartialFields2{Password: "password"}).Post(uri)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode())
		}
	})
}

func getAuthCookie(authManager *auth.Manager, cookies []*http.Cookie) (uuid.UUID, error) {
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "Authorization" {
			authCookie = cookie
		}
	}
	if authCookie == nil {
		return uuid.UUID{}, errors.New("expected Authorization cookie")
	}
	id, err := authManager.Parse(authCookie.Value)
	if err != nil {
		return uuid.UUID{}, err
	}
	return id, nil
}
