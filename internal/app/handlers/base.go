package handlers

import (
	"fmt"
	"gopkg.in/telebot.v4"
	"log/slog"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
)

type Base struct {
	log *logger.Logger
	us  *services.Users
}

func NewBase(log *logger.Logger, us *services.Users) *Base {
	return &Base{
		log: log,
		us:  us,
	}
}

func (b *Base) StartHandler(c telebot.Context) error {
	btns := getReplyButtons(c)
	return c.Send(fmt.Sprintf("👋 Добро пожаловать, %s!", c.Sender().FirstName), btns)
}

func (b *Base) AcceptOfferHandler(c telebot.Context) error {
	btns := getReplyButtons(c)

	err := b.us.UpdateIsSign(c.Sender().ID, true)
	if err != nil {
		b.log.Error("Failed to update sign", err, slog.Int64("userId", c.Sender().ID))
		return c.Send(constants.UserError, btns)
	}

	return c.Send(fmt.Sprintf("👋 Добро пожаловать, %s!", c.Sender().FirstName), btns)
}

func (b *Base) OnTextHandler(c telebot.Context) error {
	btns := getReplyButtons(c)
	return c.Send("🤔 Неизвестная команда. Используйте /help для получения списка команд", btns)
}

func (b *Base) InfoHandler(c telebot.Context) error {
	btns := getReplyButtons(c)
	return c.Send("💡 Информация", btns)
}
