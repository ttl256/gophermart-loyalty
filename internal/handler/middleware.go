package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

type ctxKey int

const (
	userIDKey ctxKey = iota
)

func (h *HTTPHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.JWT == nil {
			h.Logger.Error("jwt manager is nil")
			hErr := http.StatusInternalServerError
			http.Error(w, http.StatusText(hErr), hErr)
			return
		}
		cookie, err := r.Cookie("Authorization")
		if err != nil {
			hErr := http.StatusUnauthorized
			http.Error(w, http.StatusText(hErr), hErr)
			return
		}
		id, err := h.JWT.Parse(cookie.Value)
		if err != nil {
			h.Logger.Debug("parsing jwt", slog.Any("error", err))
			hErr := http.StatusUnauthorized
			http.Error(w, http.StatusText(hErr), hErr)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(userIDKey)
	id, ok := v.(uuid.UUID)
	return id, ok
}
