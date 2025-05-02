package middleware

import (
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/repository"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
)

type Users struct {
	log *logger.Logger
	ur  *repository.Users
	ss  *services.Subscriptions
}

func NewUsers(log *logger.Logger, ur *repository.Users, ss *services.Subscriptions) *Users {
	return &Users{
		log: log,
		ur:  ur,
		ss:  ss,
	}
}

func (u *Users) IsUser(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		data, err := u.ur.Get(c.Sender().ID)
		if err != nil {
			u.log.Error("Error while fetching user", err)
		}

		if data == nil {
			return next(c)
		}

		if data.Username != c.Sender().Username || data.Firstname != c.Sender().FirstName || data.Lastname != c.Sender().LastName {
			err = u.ur.Update(c.Sender().ID, data)
			if err != nil {
				u.log.Error("Error while updating user", err)
			}
		}

		return next(c)
	}
}
