package constants

import (
	"errors"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds on the user's balance")
	ErrUserNotFound      = errors.New("user not found")

	ErrEmptyFields = errors.New("one of the fields is empty")
)
