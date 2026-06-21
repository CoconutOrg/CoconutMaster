package types

import "errors"

var (
	ErrNotFound           = errors.New("Not found!")
	ErrInvalidCredentials = errors.New("Invalid credentials!")
	ErrAlreadyExists      = errors.New("Data must be unique!")
)
