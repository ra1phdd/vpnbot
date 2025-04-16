package handlers

import (
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
)

type Subscriptions struct {
	log     *logger.Logger
	subBtns *services.Buttons
}

func NewSubscriptions(log *logger.Logger, subBtns *services.Buttons) *Subscriptions {
	return &Subscriptions{
		log:     log,
		subBtns: subBtns,
	}
}

func (s *Subscriptions) ChooseDurationHandler(c telebot.Context) error {
	return c.Send("Выберите подписку:", s.subBtns.AddBtns())
}
