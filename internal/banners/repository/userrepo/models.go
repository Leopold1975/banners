package userrepo

import "errors"

var (
	ErrNotFound      = errors.New("user not found")
	ErrAleradyExists = errors.New("user already exists")
)
