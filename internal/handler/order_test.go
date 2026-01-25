package handler_test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	"github.com/ttl256/gophermart-loyalty/internal/auth"
	"github.com/ttl256/gophermart-loyalty/internal/database"
	"github.com/ttl256/gophermart-loyalty/internal/domain"
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
	orderNumberSize    int
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
	s.orderNumberSize = 20
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

	resp, err = s.client.R().SetBody(s.validOrderNumber).SetContentType("text/plain").Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusAccepted, resp.StatusCode())

	resp, err = s.client.R().SetBody(s.validOrderNumber).SetContentType("text/plain").Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())
}

func (s *OrderSuite) TestCreateInvalidOrder() {
	login, password := rand.Text(), rand.Text()
	registerReq := handler.RegisterRequest{Login: login, Password: password}
	resp, err := s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	resp, err = s.client.R().SetBody(s.invalidOrderNumber).SetContentType("text/plain").Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusUnprocessableEntity, resp.StatusCode())
}

func (s *OrderSuite) TestCreateExistingOrder() {
	login, password := rand.Text(), rand.Text()
	registerReq := handler.RegisterRequest{Login: login, Password: password}
	resp, err := s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	resp, err = s.client.R().SetBody(s.validOrderNumber).SetContentType("text/plain").Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusAccepted, resp.StatusCode())

	login, password = rand.Text(), rand.Text()
	registerReq = handler.RegisterRequest{Login: login, Password: password}
	resp, err = s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	resp, err = s.client.R().SetBody(s.validOrderNumber).SetContentType("text/plain").Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusConflict, resp.StatusCode())
}

func (s *OrderSuite) TestUnauthorized() {
	resp, err := s.client.R().SetBody(s.validOrderNumber).Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusUnauthorized, resp.StatusCode())

	resp, err = s.client.R().Get("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusUnauthorized, resp.StatusCode())
}

func (s *OrderSuite) TestBadRequest() {
	login, password := rand.Text(), rand.Text()
	registerReq := handler.RegisterRequest{Login: login, Password: password}
	resp, err := s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	resp, err = s.client.R().SetBody(s.validOrderNumber).SetContentType("application/json").Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusBadRequest, resp.StatusCode())

	resp, err = s.client.R().SetBody("").SetContentType("text/plain").Post("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusBadRequest, resp.StatusCode())
}

func (s *OrderSuite) TestGetNoOrders() {
	login, password := rand.Text(), rand.Text()
	registerReq := handler.RegisterRequest{Login: login, Password: password}
	resp, err := s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	resp, err = s.client.R().Get("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusNoContent, resp.StatusCode())
}

func (s *OrderSuite) TestOrdersSortedByTime() {
	login, password := rand.Text(), rand.Text()
	registerReq := handler.RegisterRequest{Login: login, Password: password}
	resp, err := s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	const numOrders = 10
	orderNumbers := make([]domain.OrderNumber, 0, numOrders)
	for range numOrders {
		var order string
		order, err = generateLuhn(s.orderNumberSize)
		s.Require().NoError(err)
		resp, err = s.client.R().SetBody(order).SetContentType("text/plain").Post("/api/user/orders")
		s.Require().NoError(err)
		s.Equal(http.StatusAccepted, resp.StatusCode())
		orderNumbers = append(orderNumbers, domain.OrderNumber(order))
	}

	var orderResponse []handler.OrderResponse
	resp, err = s.client.R().SetResult(&orderResponse).Get("/api/user/orders")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	s.Len(orderResponse, len(orderNumbers))

	orderResponseNumbers := make([]domain.OrderNumber, 0, len(orderResponse))
	for i := len(orderResponse) - 1; i >= 0; i-- {
		orderResponseNumbers = append(orderResponseNumbers, orderResponse[i].Number)
	}
	s.Equal(orderNumbers, orderResponseNumbers)
}

func (s *OrderSuite) TestGetBalanceFromEmpty() {
	login, password := rand.Text(), rand.Text()
	registerReq := handler.RegisterRequest{Login: login, Password: password}
	resp, err := s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	want, err := json.Marshal(handler.BalanceResponse{
		Current:   handler.Money(decimal.Zero),
		Withdrawn: handler.Money(decimal.Zero),
	})
	s.Require().NoError(err)

	resp, err = s.client.R().Get("/api/user/balance")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())
	s.Equal(string(want), string(resp.Bytes()))
}

func (s *OrderSuite) TestGetBalanceNoWithdrawals() {
	login, password := rand.Text(), rand.Text()
	registerReq := handler.RegisterRequest{Login: login, Password: password}
	resp, err := s.client.R().SetBody(registerReq).Post("/api/user/register")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	authCookie, err := getAuthCookie(resp.Cookies())
	s.Require().NoError(err)
	id, err := s.jwt.Parse(authCookie.Value)
	s.Require().NoError(err)

	queries := database.New(s.pool)
	var total decimal.Decimal
	for i := 1; i < 10; i++ {
		var (
			number  string
			accrual decimal.Decimal
		)
		number, err = generateLuhn(s.orderNumberSize)
		s.Require().NoError(err)
		accrual, err = decimal.NewFromString(fmt.Sprintf("%[1]d.%[1]d%[1]d", i))
		s.Require().NoError(err)
		total = total.Add(accrual)
		_, err = queries.InsertOrder(s.ctx, database.InsertOrderParams{
			Number:  number,
			UserID:  id,
			Status:  domain.OrderStatusPROCESSED.String(),
			Accrual: accrual,
		})
		s.Require().NoError(err)
	}

	want, err := json.Marshal(
		handler.BalanceResponse{
			Current:   handler.Money(total),
			Withdrawn: handler.Money(decimal.Zero),
		},
	)
	s.Require().NoError(err)

	resp, err = s.client.R().Get("/api/user/balance")
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())
	s.Equal(string(want), string(resp.Bytes()))
}

func generateLuhn(size int) (string, error) {
	if size < 2 {
		return "", fmt.Errorf("size must be >= 2, got %d", size)
	}

	digits := make([]int, size)

	for i := range size {
		d, err := randDigit()
		if err != nil {
			return "", err
		}
		digits[i] = d
	}

	if digits[0] == 0 {
		d, err := randDigitNonZero()
		if err != nil {
			return "", err
		}
		digits[0] = d
	}

	digits[size-1] = luhnCheckDigit(digits[:size-1])

	out := make([]byte, size)
	for i, d := range digits {
		out[i] = byte('0' + d)
	}
	return string(out), nil
}

func randDigit() (int, error) {
	var b [1]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return int(b[0] % 10), nil
}

func randDigitNonZero() (int, error) {
	var b [1]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return int(b[0]%9) + 1, nil
}

func luhnCheckDigit(payload []int) int {
	sum := 0
	double := true
	for i := len(payload) - 1; i >= 0; i-- {
		d := payload[i]
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return (10 - (sum % 10)) % 10
}
