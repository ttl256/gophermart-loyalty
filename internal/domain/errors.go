package domain

import "errors"

var (
	ErrLoginExists        = errors.New("login is taken")
	ErrInvalidCredentials = errors.New("invalid credentials")

	ErrMalformedOrderNumber       = errors.New("malformed order number")
	ErrOrderAlreadyUploadedByUser = errors.New("order already uploaded by user")
	ErrOrderOwnedByAnotherUser    = errors.New("order owned by another user")

	ErrNotEnoughFunds = errors.New("not enough funds")
)
