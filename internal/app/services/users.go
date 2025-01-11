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

func NewUsers(ur *repository.Users) *Users {
	return &Users{
		ur: ur,
	}
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

func (u *Users) IsAdmin(id int64) (bool, error) {
	user, err := u.ur.GetById(id)
	if err != nil {
		return false, err
	}

	return user.IsAdmin, nil
}

func (u *Users) IsSign(id int64) (bool, error) {
	user, err := u.ur.GetById(id)
	if err != nil {
		return false, err
	}

	return user.IsSign, nil
}

func (u *Users) Add(user models.User) error {
	if user.ID == 0 {
		return constants.ErrUserNotFound
	}

	return u.ur.Add(user)
}

func (u *Users) UpdateSign(id int64, value bool) error {
	user, err := u.ur.GetById(id)
	if err != nil {
		return err
	}

	if user.IsSign == value {
		return nil
	}
	user.IsSign = value

	return u.ur.Update(user)
}
