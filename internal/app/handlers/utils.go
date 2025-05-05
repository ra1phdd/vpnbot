package handlers

import (
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"time"
)

func getReplyButtons(c telebot.Context) *telebot.ReplyMarkup {
	if btns, ok := c.Get("replyKeyboard").(*telebot.ReplyMarkup); ok {
		return btns
	}
	return &telebot.ReplyMarkup{}
}

func getUser(c telebot.Context, us *services.Users) *models.User {
	if user, ok := c.Get("user").(*models.User); ok {
		return user
	}

	user, err := us.Get(c.Sender().ID)
	if err != nil {
		return nil
	}
	return user
}

func getSubscription(c telebot.Context, ss *services.Subscriptions) *models.Subscription {
	if sub, ok := c.Get("sub").(*models.Subscription); ok {
		return sub
	}

	sub, err := ss.GetLastByUserID(c.Sender().ID, true)
	if err != nil {
		return nil
	}
	return sub
}

func validateSubscription(c telebot.Context, ss *services.Subscriptions) error {
	sub, ok := c.Get("sub").(*models.Subscription)
	if !ok {
		var err error
		sub, err = ss.GetLastByUserID(c.Sender().ID, true)
		if err != nil {
			return c.Send(constants.UserHasNoRights, getReplyButtons(c))
		}
	}

	if !sub.IsActive && sub.EndDate.Before(time.Now()) && (!sub.EndDate.IsZero() || sub.ID == 0) {
		return c.Send(constants.UserHasNoRights, getReplyButtons(c))
	}
	return nil
}
