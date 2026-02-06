package auth

import (
	"fmt"

	"github.com/alexedwards/argon2id"
)

type PasswordHash string

func NewHashPassword(password string) (PasswordHash, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return PasswordHash(hash), nil
}

func (h PasswordHash) ComparePassword(password string) (bool, error) {
	ok, err := argon2id.ComparePasswordAndHash(password, string(h))
	if err != nil {
		return false, fmt.Errorf("comparing password and hash: %w", err)
	}
	return ok, nil
}
