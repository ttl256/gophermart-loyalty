package handler

import "github.com/google/uuid"

// HealthStatus ENUM(ok).
type HealthStatus int //nolint: recvcheck //fine

type HealthResponse struct {
	Status HealthStatus `json:"status"`
}

type HealthResponseWithID struct {
	Status HealthStatus `json:"status"`
	ID     uuid.UUID    `json:"id"`
}
