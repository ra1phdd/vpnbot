package handlers

import (
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"time"
)

func getReplyButtons(c telebot.Context) *telebot.ReplyMarkup {
	if btns, ok := c.Get("replyKeyboard").(*telebot.ReplyMarkup); ok {
		return btns
	}
	return &telebot.ReplyMarkup{}
}

func validateSubscription(c telebot.Context) error {
	if sub, ok := c.Get("sub").(*models.Subscription); ok {
		if !sub.IsActive && sub.EndDate.Before(time.Now()) && (!sub.EndDate.IsZero() || sub.ID == 0) {
			return c.Send(constants.UserHasNoRights, getReplyButtons(c))
		}
	}
	return nil
}
