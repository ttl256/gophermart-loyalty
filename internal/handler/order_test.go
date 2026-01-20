package handler_test

import (
	"context"
	"crypto/rand"
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

func TestOrder(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(OrderSuite))
}

type OrderSuite struct {
	suite.Suite

	ctx                context.Context
	pg                 *testutil.PG
	repo               *repository.DBStorage
	pool               *pgxpool.Pool
	jwt                *auth.Manager
	server             *httptest.Server
	client             *resty.Client
	validOrderNumber   string
	invalidOrderNumber string
}

func (s *OrderSuite) SetupSuite() {
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
	orderSvc := service.NewOrderService(repo)

	h := handler.HTTPHandler{
		AuthService:  authSvc,
		OrderService: orderSvc,
		JWT:          authManager,
		Logger:       slog.Default(),
	}
	srv := httptest.NewServer(h.Routes())
	s.server = srv

	s.validOrderNumber = "49927398716"
	s.invalidOrderNumber = "49927398717"
}

func (s *OrderSuite) TearDownSuite() {
	s.Require().NoError(s.client.Close())
	s.server.Close()
	s.pool.Close()
	s.repo.Close()
	s.Require().NoError(s.pg.Close(s.ctx))
}

func (s *OrderSuite) SetupTest() {
	s.Require().NoError(s.repo.Migrate(repository.MigrationActionUp))
	client := resty.New().SetBaseURL(s.server.URL)
	s.client = client
}

func (s *OrderSuite) TearDownTest() {
	s.Require().NoError(s.repo.Migrate(repository.MigrationActionDrop))
}

func (s *OrderSuite) TestCreateValidOrder() {
	login, password := rand.Text(), rand.Text()
	registerReq := handler.RegisterRequest{Login: login, Password: password}
	resp, err := s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	resp, err = s.client.R().SetBody(s.validOrderNumber).Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusAccepted, resp.StatusCode())

	resp, err = s.client.R().SetBody(s.validOrderNumber).Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())
}

func (s *OrderSuite) TestCreateInvalidOrder() {
	login, password := rand.Text(), rand.Text()
	registerReq := handler.RegisterRequest{Login: login, Password: password}
	resp, err := s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	resp, err = s.client.R().SetBody(s.invalidOrderNumber).Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusUnprocessableEntity, resp.StatusCode())
}

func (s *OrderSuite) TestCreateExistingOrder() {
	login, password := rand.Text(), rand.Text()
	registerReq := handler.RegisterRequest{Login: login, Password: password}
	resp, err := s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	resp, err = s.client.R().SetBody(s.validOrderNumber).Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusAccepted, resp.StatusCode())

	login, password = rand.Text(), rand.Text()
	registerReq = handler.RegisterRequest{Login: login, Password: password}
	resp, err = s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	resp, err = s.client.R().SetBody(s.validOrderNumber).Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusConflict, resp.StatusCode())
}

func (s *OrderSuite) TestUnauthorized() {
	resp, err := s.client.R().SetBody(s.validOrderNumber).Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusUnauthorized, resp.StatusCode())
}
