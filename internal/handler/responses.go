package handler

// HealthStatus ENUM(ok).
type HealthStatus int //nolint: recvcheck //fine

type HealthResponse struct {
	Status HealthStatus `json:"status"`
}
