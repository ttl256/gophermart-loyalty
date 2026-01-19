package domain

import "github.com/google/uuid"

type User struct {
	ID    uuid.UUID
	Login string
}

func NewUser(login string) User {
	return User{
		ID:    uuid.New(),
		Login: login,
	}
}
