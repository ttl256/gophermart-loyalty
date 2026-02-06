package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/ttl256/gophermart-loyalty/internal/auth"
	"github.com/ttl256/gophermart-loyalty/internal/domain"
)

type AuthService interface {
	RegisterUser(ctx context.Context, user domain.User, password string) (uuid.UUID, error)
	LoginUser(ctx context.Context, login string, password string) (domain.User, error)
}

type OrderService interface {
	RegisterOrder(ctx context.Context, userID uuid.UUID, order string) (uuid.UUID, error)
	GetOrders(ctx context.Context, userID uuid.UUID) ([]domain.Order, error)
	GetBalance(ctx context.Context, userID uuid.UUID) (domain.Balance, error)
	Withdraw(ctx context.Context, userID uuid.UUID, order string, sum decimal.Decimal) error
	GetWithdrawals(ctx context.Context, userID uuid.UUID) ([]domain.Withdrawal, error)
}

type HTTPHandler struct {
	JWT          *auth.Manager
	AuthService  AuthService
	OrderService OrderService
	Logger       *slog.Logger
}

func (h *HTTPHandler) Routes() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/healthz", h.HealthHandler)
	r.Post("/api/user/register", h.RegisterHandler)
	r.Post("/api/user/login", h.LoginHandler)

	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Post("/api/user/orders", h.UploadOrder)
		r.Get("/api/user/orders", h.GetOrders)
		r.Get("/api/user/balance", h.GetBalance)
		r.Post("/api/user/balance/withdraw", h.Withdraw)
		r.Get("/api/user/withdrawals", h.GetWithdrawals)
	})

	return r
}

func (h *HTTPHandler) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	data, err := json.Marshal(HealthResponse{Status: HealthStatusOk})
	if err != nil {
		h.Logger.Error("", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
func (h *HTTPHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.Logger.Debug("bad request", slog.Any("error", err))
		hErr := http.StatusBadRequest
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	if err = req.Validate(); err != nil {
		h.Logger.Debug("bad request", slog.Any("error", err))
		hErr := http.StatusBadRequest
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	id, err := h.AuthService.RegisterUser(r.Context(), domain.NewUser(req.Login), req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrLoginExists) {
			h.Logger.Debug("register user", slog.Any("error", err))
			hErr := http.StatusConflict
			http.Error(w, http.StatusText(hErr), hErr)
			return
		}
		h.Logger.Error("register user", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	err = h.SetCookie(w, id)
	if err != nil {
		h.Logger.Error("issuing jwt", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *HTTPHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.Logger.Debug("bad request", slog.Any("error", err))
		hErr := http.StatusBadRequest
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	if err = req.Validate(); err != nil {
		h.Logger.Debug("bad request", slog.Any("error", err))
		hErr := http.StatusBadRequest
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	user, err := h.AuthService.LoginUser(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			h.Logger.Debug("login user", slog.Any("error", err))
			hErr := http.StatusUnauthorized
			http.Error(w, http.StatusText(hErr), hErr)
			return
		}
		h.Logger.Error("login user", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	err = h.SetCookie(w, user.ID)
	if err != nil {
		h.Logger.Error("issuing jwt", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *HTTPHandler) SetCookie(w http.ResponseWriter, id uuid.UUID) error {
	token, err := h.JWT.Issue(id)
	if err != nil {
		return fmt.Errorf("issuing jwt: %w", err)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "Authorization",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
	})
	return nil
}

func (h *HTTPHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		h.Logger.Error("parsing content-type", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	if contentType != "text/plain" || r.ContentLength == 0 {
		h.Logger.Debug("invalid request body")
		hErr := http.StatusBadRequest
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) //nolint: mnd //fine
	data, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Error("reading request", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	_, err = h.OrderService.RegisterOrder(r.Context(), id, string(bytes.TrimSpace(data)))
	if err != nil {
		if errors.Is(err, domain.ErrMalformedOrderNumber) {
			h.Logger.Debug("malformed order number", slog.Any("error", err))
			hErr := http.StatusUnprocessableEntity
			http.Error(w, http.StatusText(hErr), hErr)
			return
		}
		if errors.Is(err, domain.ErrOrderAlreadyUploadedByUser) {
			w.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, domain.ErrOrderOwnedByAnotherUser) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		h.Logger.Error("registering order", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *HTTPHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	orders, err := h.OrderService.GetOrders(r.Context(), id)
	if err != nil {
		h.Logger.Error("getting orders", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	if len(orders) == 0 {
		h.Logger.Debug("no orders")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	resp := make([]OrderResponse, 0, len(orders))
	for _, i := range orders {
		resp = append(resp, OrderResponse{
			Number:     i.Number,
			Status:     i.Status,
			Accrual:    Money(i.Accrual),
			UploadedAt: i.UploadedAt,
		})
	}
	data, err := json.Marshal(resp)
	if err != nil {
		h.Logger.Error("encoding json", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *HTTPHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	balance, err := h.OrderService.GetBalance(r.Context(), id)
	if err != nil {
		h.Logger.Error("getting balance", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	balanceResponse := BalanceResponse{
		Current:   Money(balance.Current),
		Withdrawn: Money(balance.Withdrawn),
	}
	data, err := json.Marshal(balanceResponse)
	if err != nil {
		h.Logger.Error("encoding json", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *HTTPHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	var req WithdrawalRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.Logger.Debug("bad request", slog.Any("error", err))
		hErr := http.StatusBadRequest
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	err = h.OrderService.Withdraw(r.Context(), id, req.Order, decimal.Decimal(req.Sum))
	if err != nil {
		if errors.Is(err, domain.ErrMalformedOrderNumber) {
			h.Logger.Debug("malformed order number", slog.Any("error", err))
			hErr := http.StatusUnprocessableEntity
			http.Error(w, http.StatusText(hErr), hErr)
			return
		}
		if errors.Is(err, domain.ErrNotEnoughFunds) {
			h.Logger.Debug("not enough funds", slog.Any("error", err))
			hErr := http.StatusPaymentRequired
			http.Error(w, http.StatusText(hErr), hErr)
			return
		}
		h.Logger.Error("", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *HTTPHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	withdrawals, err := h.OrderService.GetWithdrawals(r.Context(), id)
	if err != nil {
		h.Logger.Error("getting withdrawals", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	withdrawalResponse := make([]WithdrawalsResponse, 0, len(withdrawals))
	for _, i := range withdrawals {
		withdrawalResponse = append(withdrawalResponse, WithdrawalsResponse{
			Order:       i.Order,
			Sum:         Money(i.Sum),
			ProcessedAt: i.ProcessedAt,
		})
	}
	data, err := json.Marshal(withdrawalResponse)
	if err != nil {
		h.Logger.Error("", slog.Any("error", err))
		hErr := http.StatusInternalServerError
		http.Error(w, http.StatusText(hErr), hErr)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
