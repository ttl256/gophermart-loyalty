package handler_test

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	"github.com/ttl256/gophermart-loyalty/internal/auth"
	"github.com/ttl256/gophermart-loyalty/internal/handler"
	"github.com/ttl256/gophermart-loyalty/internal/logger"
	"github.com/ttl256/gophermart-loyalty/internal/repository"
	"github.com/ttl256/gophermart-loyalty/internal/service"
	"github.com/ttl256/gophermart-loyalty/internal/testutil"
	"resty.dev/v3"
)

func TestAuth(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AuthSuite))
}

type AuthSuite struct {
	suite.Suite

	ctx    context.Context
	pg     *testutil.PG
	repo   *repository.DBStorage
	pool   *pgxpool.Pool
	jwt    *auth.Manager
	server *httptest.Server
	client *resty.Client
}

func (s *AuthSuite) SetupSuite() {
	s.ctx = context.Background()
	pg, err := testutil.StartPG(s.ctx)
	s.Require().NoError(err)
	s.pg = pg

	logger.Initialize(slog.LevelInfo)

	repo, err := repository.NewDBStorage(s.ctx, pg.DSN)
	s.Require().NoError(err)
	s.Require().NoError(repo.RepoPing(s.ctx))
	s.repo = repo

	s.pool, err = pgxpool.New(s.ctx, s.pg.DSN)
	s.Require().NoError(err)

	authSvc := service.NewAuthService(repo)
	authManager := auth.NewManager("test", 1*time.Hour)
	s.jwt = authManager

	h := handler.HTTPHandler{
		AuthService:  authSvc,
		OrderService: nil,
		JWT:          authManager,
		Logger:       slog.Default(),
	}
	srv := httptest.NewServer(h.Routes())
	s.server = srv
}

func (s *AuthSuite) TearDownSuite() {
	s.Require().NoError(s.client.Close())
	s.server.Close()
	s.pool.Close()
	s.repo.Close()
	s.Require().NoError(s.pg.Close(s.ctx))
}

func (s *AuthSuite) SetupTest() {
	s.Require().NoError(s.repo.Migrate(repository.MigrationActionUp))
	client := resty.New().SetBaseURL(s.server.URL)
	s.client = client
}

func (s *AuthSuite) TearDownTest() {
	s.Require().NoError(s.repo.Migrate(repository.MigrationActionDrop))
}

func (s *AuthSuite) TestRegisterAndLogin() {
	login, password := rand.Text(), rand.Text()

	registerReq := handler.RegisterRequest{Login: login, Password: password}
	resp, err := s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())
	cookie, err := getAuthCookie(resp.Cookies())
	s.Require().NoError(err)
	registerUUID, err := s.jwt.Parse(cookie.Value)
	s.Require().NoError(err)

	loginReq := registerReq
	resp, err = s.client.R().SetBody(loginReq).Post("/api/user/login")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())
	cookie, err = getAuthCookie(resp.Cookies())
	s.Require().NoError(err)
	loginUUID, err := s.jwt.Parse(cookie.Value)
	s.Require().NoError(err)

	s.Equal(
		registerUUID,
		loginUUID,
		"auth cookie must contain the same user ID for register and login responses",
	)
}

func (s *AuthSuite) TestInvalidPassword() {
	login, password := rand.Text(), rand.Text()

	registerReq := handler.RegisterRequest{Login: login, Password: password}
	resp, err := s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	loginReq := handler.RegisterRequest{Login: login, Password: rand.Text()}
	resp, err = s.client.R().SetBody(loginReq).Post("/api/user/login")
	s.Require().NoError(err)
	s.Equal(http.StatusUnauthorized, resp.StatusCode())
}

func (s *AuthSuite) TestLoginWithNonExistentLogin() {
	login, password := rand.Text(), rand.Text()

	loginReq := handler.RegisterRequest{Login: login, Password: password}
	resp, err := s.client.R().SetBody(loginReq).Post("/api/user/login")
	s.Require().NoError(err)
	s.Equal(http.StatusUnauthorized, resp.StatusCode())
}

func (s *AuthSuite) TestRegisterConflict() {
	login := rand.Text()

	registerReq := handler.RegisterRequest{Login: login, Password: rand.Text()}
	resp, err := s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	registerReq = handler.RegisterRequest{Login: login, Password: rand.Text()}
	resp, err = s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusConflict, resp.StatusCode())
}

func (s *AuthSuite) TestReqEmptyFields() {
	emptyRequests := []handler.RegisterRequest{
		{Login: "", Password: rand.Text()},
		{Login: rand.Text(), Password: ""},
		{Login: "", Password: ""},
	}
	for _, req := range emptyRequests {
		resp, err := s.client.R().SetBody(req).Post("/api/user/register")
		s.Require().NoError(err)
		s.Equal(http.StatusBadRequest, resp.StatusCode())
		resp, err = s.client.R().SetBody(req).Post("/api/user/login")
		s.Require().NoError(err)
		s.Equal(http.StatusBadRequest, resp.StatusCode())
	}
}

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
