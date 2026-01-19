package handler

import "errors"

var errEmptyFields = errors.New("empty fields")

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (r RegisterRequest) Validate() error {
	if r.Login == "" || r.Password == "" {
		return errEmptyFields
	}
	return nil
}
