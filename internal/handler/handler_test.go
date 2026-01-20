package handler_test

import (
	"errors"
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

// func TestAuthMiddleware(t *testing.T) {
// 	ctx := context.Background()
// 	pg, err := testutil.StartPG(ctx)
// 	require.NoError(t, err)
// 	t.Cleanup(func() {
// 		errClose := pg.Close(ctx)
// 		if errClose != nil {
// 			t.Logf("terminating postgres container: %v", err)
// 		}
// 	})

// 	_ = logger.Initialize(slog.LevelInfo)

// 	repo, err := repository.NewDBStorage(ctx, pg.DSN)
// 	require.NoError(t, err)
// 	t.Cleanup(repo.Close)
// 	require.NoError(t, repo.RepoPing(ctx))
// 	require.NoError(t, repo.Migrate())

// 	authSvc := service.NewAuthService(repo)
// 	authManager := auth.NewManager("test", 1*time.Hour)

// 	h := handler.HTTPHandler{
// 		AuthService:  authSvc,
// 		OrderService: nil,
// 		JWT:          authManager,
// 		Logger:       slog.Default(),
// 	}
// 	srv := httptest.NewServer(h.Routes())
// 	t.Cleanup(srv.Close)

// 	client := resty.New().SetBaseURL(srv.URL).SetCookieJar(nil)
// 	t.Cleanup(func() {
// 		errClient := client.Close()
// 		if errClient != nil {
// 			t.Logf("closing http client: %v", err)
// 		}
// 	})

// 	t.Run("registration and login", func(t *testing.T) {
// 		var (
// 			resp         *resty.Response
// 			registerUUID uuid.UUID
// 			authCookie   *http.Cookie
// 		)
// 		for _, uri := range []string{"/api/user/register", "/api/user/login"} {
// 			t.Run(uri, func(t *testing.T) {
// 				registerReq := handler.RegisterRequest{Login: "user1", Password: "passwd1"}
// 				resp, err = client.R().SetBody(registerReq).Post(uri)
// 				require.NoError(t, err)
// 				assert.Equal(t, http.StatusOK, resp.StatusCode())

// 				authCookie, err = getAuthCookie(resp.Cookies())
// 				require.NoError(t, err, "unable to get Authorization cookie")
// 				registerUUID, err = authManager.Parse(authCookie.Value)
// 				require.NoError(t, err, "unable to parse JWT cookie")

// 				var body handler.HealthResponseWithID
// 				resp, err = client.R().SetResult(&body).Get("/healthzp")
// 				require.NoError(t, err)
// 				assert.Equal(t, http.StatusUnauthorized, resp.StatusCode())

// 				resp, err = client.R().SetCookie(authCookie).SetResult(&body).Get("/healthzp")
// 				require.NoError(t, err)
// 				assert.Equal(t, http.StatusOK, resp.StatusCode())

// 				assert.Equal(t, registerUUID, body.ID, "User ID from cookie and healthzp don't match")
// 			})
// 		}
// 	})
// }

func getAuthCookie(cookies []*http.Cookie) (*http.Cookie, error) {
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "Authorization" {
			authCookie = cookie
			break
		}
	}
	if authCookie == nil {
		return nil, errors.New("expected Authorization cookie")
	}
	return authCookie, nil
}

// func TestRegisterOrder(t *testing.T) {
// 	ctx := context.Background()
// 	pg, err := testutil.StartPG(ctx)
// 	require.NoError(t, err)
// 	t.Cleanup(func() {
// 		errClose := pg.Close(ctx)
// 		if errClose != nil {
// 			t.Logf("terminating postgres container: %v", err)
// 		}
// 	})

// 	_ = logger.Initialize(slog.LevelInfo)

// 	repo, err := repository.NewDBStorage(ctx, pg.DSN)
// 	require.NoError(t, err)
// 	t.Cleanup(repo.Close)
// 	require.NoError(t, repo.RepoPing(ctx))
// 	require.NoError(t, repo.Migrate())

// 	authSvc := service.NewAuthService(repo)
// 	authManager := auth.NewManager("test", 1*time.Hour)

// 	orderSvc := service.NewOrderService(repo)

// 	h := handler.HTTPHandler{
// 		AuthService:  authSvc,
// 		OrderService: orderSvc,
// 		JWT:          authManager,
// 		Logger:       slog.Default(),
// 	}
// 	srv := httptest.NewServer(h.Routes())
// 	t.Cleanup(srv.Close)

// 	client := resty.New().SetBaseURL(srv.URL)
// 	t.Cleanup(func() {
// 		errClient := client.Close()
// 		if errClient != nil {
// 			t.Logf("closing http client: %v", err)
// 		}
// 	})
// 	registerReq := handler.RegisterRequest{Login: "user1", Password: "passwd1"}
// 	resp, err := client.R().SetBody(registerReq).Post("/api/user/register")
// 	require.NoError(t, err)
// 	assert.Equal(t, http.StatusOK, resp.StatusCode())

// 	resp, err = client.R().SetBody("49927398716").Post("/api/user/orders")
// 	require.NoError(t, err)
// 	assert.Equal(t, http.StatusAccepted, resp.StatusCode())

// 	resp, err = client.R().SetBody("49927398716").Post("/api/user/orders")
// 	require.NoError(t, err)
// 	assert.Equal(t, http.StatusOK, resp.StatusCode())

// 	resp, err = client.R().SetBody("49927398717").Post("/api/user/orders")
// 	require.NoError(t, err)
// 	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode())

// 	registerReq = handler.RegisterRequest{Login: "user2", Password: "passwd1"}
// 	resp, err = client.R().SetBody(registerReq).Post("/api/user/register")
// 	require.NoError(t, err)
// 	assert.Equal(t, http.StatusOK, resp.StatusCode())

// 	resp, err = client.R().SetBody("49927398716").Post("/api/user/orders")
// 	require.NoError(t, err)
// 	assert.Equal(t, http.StatusConflict, resp.StatusCode())
// }
