package base

import (
	"gopkg.in/telebot.v3"
	"nsvpn/internal/app/models"
)

type Base interface {
}

type Endpoint struct {
	Base Base
}

func (e *Endpoint) AcceptOfferHandler(c telebot.Context) error {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true, IsPersistent: true}

	btns := make(map[string]telebot.Btn, 1)
	btns[models.AcceptOfferButton.Value] = menu.Text(models.AcceptOfferButton.Display)

	menu.Inline(
		menu.Row(btns["accept"]),
	)

	err := c.Send("Чтобы начать пользоваться NSVPN, необходимо принять условия публичной [оферты](https://teletype.in/@nsvpn/Dpvwcj7llQx).", menu)
	if err != nil {
		return err
	}
	return nil
}

func (e *Endpoint) StartHandler(c telebot.Context) error {
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true, IsPersistent: true}

	btns := make(map[string]telebot.Btn, len(models.ClientButtons))
	for _, item := range models.ClientButtons {
		btns[item.Value] = menu.Text(item.Display)
	}

	menu.Reply(
		menu.Row(btns["attachvpn"]),
		menu.Row(btns["profile"], btns["info"]),
	)

	err := c.Send("Чтобы начать пользоваться NSVPN, необходимо принять условия публичной [оферты](https://teletype.in/@nsvpn/Dpvwcj7llQx).", menu)
	if err != nil {
		return err
	}
	return nil
}

func (e *Endpoint) HelpHandler(c telebot.Context) error {
	err := c.Send("🚀 Базовые команды\n/help - Посмотреть справку о командах\n")
	if err != nil {
		return err
	}
	return nil
}
