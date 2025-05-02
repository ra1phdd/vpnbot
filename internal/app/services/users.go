package services

import (
	"log/slog"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
)

type Users struct {
	log *logger.Logger
	ur  *repository.Users
}

func NewUsers(log *logger.Logger, ur *repository.Users) *Users {
	return &Users{
		log: log,
		ur:  ur,
	}
}

func (us *Users) Get(id int64) (user *models.User, err error) {
	if id == 0 {
		return nil, constants.ErrEmptyFields
	}

	return us.ur.Get(id)
}

func (us *Users) Add(user *models.User) error {
	if user.ID == 0 && user.Username == "" {
		return constants.ErrEmptyFields
	}

	return us.ur.Add(user)
}

func (us *Users) Update(id int64, newUser *models.User) error {
	if id == 0 || newUser == nil {
		return constants.ErrEmptyFields
	}

	return us.ur.Update(id, newUser)
}

func (us *Users) UpdateIsAdmin(id int64, isAdmin bool) error {
	if id == 0 {
		return constants.ErrEmptyFields
	}

	return us.ur.UpdateIsAdmin(id, isAdmin)
}

func (us *Users) UpdateIsSign(id int64, isSign bool) error {
	if id == 0 {
		return constants.ErrEmptyFields
	}

	return us.ur.UpdateIsAdmin(id, isSign)
}

func (us *Users) IncrementBalance(id int64, amount int) error {
	if id == 0 || amount == 0 {
		return constants.ErrEmptyFields
	}

	return us.ur.IncrementBalance(id, amount)
}

func (us *Users) DecrementBalance(id int64, amount int) error {
	if id == 0 || amount == 0 {
		return constants.ErrEmptyFields
	}

	return us.ur.DecrementBalance(id, amount)
}

func (us *Users) Delete(id int64) error {
	if id == 0 {
		return constants.ErrEmptyFields
	}

	return us.ur.Delete(id)
}

func (us *Users) IsFound(id int64) (bool, error) {
	if id == 0 {
		return false, constants.ErrEmptyFields
	}

	data, err := us.ur.Get(id)
	if err != nil {
		us.log.Error("Error while getting user by id", err, slog.Int64("id", id))
		return false, err
	}

	if data == nil {
		return false, nil
	}
	return true, nil
}

func (us *Users) IsAdmin(id int64) (bool, error) {
	if id == 0 {
		return false, constants.ErrEmptyFields
	}

	data, err := us.ur.Get(id)
	if err != nil {
		us.log.Error("Error while getting user by id", err, slog.Int64("id", id))
		return false, err
	}

	if data == nil {
		return false, nil
	}
	return data.IsAdmin, nil
}

func (us *Users) IsSign(id int64) (bool, error) {
	if id == 0 {
		return false, constants.ErrEmptyFields
	}

	data, err := us.ur.Get(id)
	if err != nil {
		us.log.Error("Error while getting user by id", err, slog.Int64("id", id))
		return false, err
	}

	if data == nil {
		return false, nil
	}
	return data.IsSign, nil
}
