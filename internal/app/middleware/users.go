package middleware

import (
	"errors"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/repository"
	"nsvpn/internal/app/services"
)

type Users struct {
	ur *repository.Users
	ss *services.Subscriptions
}

func NewUsers(ur *repository.Users, ss *services.Subscriptions) *Users {
	return &Users{
		ur: ur,
		ss: ss,
	}
}

func (u *Users) IsUser(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		data, err := u.ur.GetById(c.Sender().ID)
		if err != nil {
			if errors.Is(err, constants.ErrUserNotFound) {
				return next(c)
			}
			return err
		}

		if data.Username != c.Sender().Username || data.Firstname != c.Sender().FirstName || data.Lastname != c.Sender().LastName {
			err = u.ur.Update(data)
			if err != nil {
				return err
			}
		}

		return next(c)
	}
}
