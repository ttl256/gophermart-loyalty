package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ttl256/gophermart-loyalty/internal/domain"
	"github.com/ttl256/gophermart-loyalty/internal/handler"
	"resty.dev/v3"
)

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e RateLimitError) Error() string {
	return fmt.Sprintf("retry after %s", e.RetryAfter)
}

// OrderStatus ENUM(REGISTERED, INVALID, PROCESSING, PROCESSED).
type OrderStatus int //nolint: recvcheck //fine

type OrderInfo struct {
	Order   domain.OrderNumber `json:"order"`
	Status  OrderStatus        `json:"status"`
	Accrual handler.Money      `json:"accrual"`
}

type Client struct {
	c *resty.Client
}

func NewClient(baseURL string) *Client {
	client := resty.New()
	client.SetBaseURL(baseURL)
	return &Client{
		c: client,
	}
}

func (c *Client) GetOrder(ctx context.Context, number domain.OrderNumber) (OrderInfo, bool, error) {
	uri, err := url.JoinPath("/api/orders", string(number))
	if err != nil {
		return OrderInfo{}, false, fmt.Errorf("joining path: %w", err)
	}
	resp, err := c.c.R().SetContext(ctx).Get(uri)
	if err != nil {
		return OrderInfo{}, false, fmt.Errorf("getting order info: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode() {
	case http.StatusOK:
		var body []byte
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return OrderInfo{}, false, fmt.Errorf("reading response body: %w", err)
		}
		var orderResp OrderInfo
		if err = json.Unmarshal(body, &orderResp); err != nil {
			return OrderInfo{}, false, fmt.Errorf("decoding response body: %w", err)
		}
		return orderResp, true, nil
	case http.StatusNoContent:
		return OrderInfo{}, false, nil
	case http.StatusTooManyRequests:
		const defaultRetryDuration = 10 * time.Second
		retryDuration := ParseRetryAfter(resp.Header().Get("Retry-After"), defaultRetryDuration)
		return OrderInfo{}, false, RateLimitError{RetryAfter: retryDuration}
	default:
		return OrderInfo{}, false, fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}
}

func ParseRetryAfter(s string, defaultDuration time.Duration) time.Duration {
	if s == "" {
		return defaultDuration
	}
	d, err := time.ParseDuration(s + "s")
	if err != nil || d < 0 {
		return defaultDuration
	}
	return d
}
