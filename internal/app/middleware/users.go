package middleware

import (
	"errors"
	tele "gopkg.in/telebot.v3"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/repository"
)

type Endpoint struct {
	b *tele.Bot
	u *repository.Users
}

func (e *Endpoint) IsUser(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		data, err := e.u.GetById(c.Sender().ID)
		if err != nil {
			if errors.Is(err, constants.ErrUserNotFound) {
				return next(c)
			}
			return err
		}

		if data.Username != c.Sender().Username || data.Firstname != c.Sender().FirstName || data.Lastname != c.Sender().LastName {
			err = e.u.Update(data)
			if err != nil {
				return err
			}
		}

		return next(c)
	}
}
