package services

import (
	"errors"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
)

type Users struct {
	ur *repository.Users
}

func NewUsers() *Users {
	return &Users{}
}

func (u *Users) IsFound(id int64) (bool, error) {
	_, err := u.ur.GetById(id)
	if err != nil {
		if errors.Is(err, constants.ErrUserNotFound) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (u *Users) Add(user models.User) error {
	if user.ID == 0 {
		return constants.ErrUserNotFound
	}

	return u.ur.Add(user)
}
