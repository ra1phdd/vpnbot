package handlers

import (
	"gopkg.in/telebot.v3"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"strconv"
)

type Base struct {
	menu *telebot.ReplyMarkup
	btns *services.Buttons
	us   *services.Users
}

func (b *Base) AcceptOfferHandler(c telebot.Context) error {
	b.btns.ReplyWithButtons(b.menu, models.ClientButtons, []int{1, 2})
	return c.Send("Чтобы начать пользоваться NSVPN, необходимо принять условия публичной [оферты](https://teletype.in/@nsvpn/Dpvwcj7llQx).", b.menu)
}

func (b *Base) StartHandler(c telebot.Context) error {
	found, err := b.us.IsFound(c.Sender().ID)
	if err != nil {
		return err
	}

	if !found {
		var partnerID *int
		partnerString := c.Data()
		*partnerID, err = strconv.Atoi(partnerString)
		if err != nil {
			return err
		}

		user := models.User{
			ID:        c.Sender().ID,
			Username:  c.Sender().Username,
			Firstname: c.Sender().FirstName,
			Lastname:  c.Sender().LastName,
			PartnerID: partnerID,
		}

		err = b.us.Add(user)
		if err != nil {
			return err
		}
	}

	b.btns.InlineWithButtons(b.menu, models.AcceptOfferButton, []int{1})
	return c.Send("Чтобы начать пользоваться NSVPN, необходимо принять условия публичной [оферты](https://teletype.in/@nsvpn/Dpvwcj7llQx).", b.menu)
}

func (b *Base) HelpHandler(c telebot.Context) error {
	err := c.Send("🚀 Базовые команды\n/help - Посмотреть справку о командах\n")
	if err != nil {
		return err
	}
	return nil
}
