package domain

import "errors"

var (
	ErrLoginExists        = errors.New("login is taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
