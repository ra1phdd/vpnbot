package services

import (
	"errors"
	"log/slog"
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

func (us *Users) IsFound(id int64) (bool, error) {
	if id == 0 {
		return false, errors.New("id is empty")
	}

	data, err := us.ur.GetById(id)
	if err != nil {
		us.log.Error("Error while getting user by id", err, slog.Int64("id", id))
		return false, err
	}

	if data == (models.User{}) {
		return false, nil
	}
	return true, nil
}

func (us *Users) IsAdmin(id int64) (bool, error) {
	if id == 0 {
		return false, errors.New("id is empty")
	}

	data, err := us.ur.GetById(id)
	if err != nil {
		us.log.Error("Error while getting user by id", err, slog.Int64("id", id))
		return false, err
	}

	if data == (models.User{}) {
		return false, nil
	}
	return data.IsAdmin, nil
}

func (us *Users) IsSign(id int64) (bool, error) {
	if id == 0 {
		return false, errors.New("id is empty")
	}

	data, err := us.ur.GetById(id)
	if err != nil {
		us.log.Error("Error while getting user by id", err, slog.Int64("id", id))
		return false, err
	}

	if data == (models.User{}) {
		return false, nil
	}
	return data.IsSign, nil
}

func (us *Users) GetById(id int64) (models.User, error) {
	if id == 0 {
		return models.User{}, errors.New("id is empty")
	}

	return us.ur.GetById(id)
}

func (us *Users) Add(user models.User) error {
	if user.ID == 0 && user.Username == "" {
		return errors.New("id or username is empty")
	}

	return us.ur.Add(user)
}

func (us *Users) Update(id int64, user models.User) error {
	if id == 0 {
		return errors.New("id is empty")
	}

	return us.ur.Update(id, user)
}

func (us *Users) Delete(id int64) error {
	if id == 0 {
		return errors.New("id is empty")
	}

	return us.ur.Delete(id)
}
