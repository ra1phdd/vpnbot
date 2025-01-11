package handlers

import (
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/services"
)

type Subscriptions struct {
	ListSubscriptions *services.Buttons
}

func NewSubscriptions(ListSubscriptions *services.Buttons) *Subscriptions {
	return &Subscriptions{
		ListSubscriptions: ListSubscriptions,
	}
}

func (s *Subscriptions) ChooseDurationHandler(c telebot.Context) error {
	return c.Send("Выберите подписку:", s.ListSubscriptions.AddBtns())
}
