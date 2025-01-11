package handlers

import (
	"fmt"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"strconv"
)

type Base struct {
	acceptOfferButtons, clientButtons *services.Buttons
	us                                *services.Users
}

func NewBase(acceptOfferButtons, clientButtons *services.Buttons, us *services.Users) *Base {
	return &Base{
		acceptOfferButtons: acceptOfferButtons,
		clientButtons:      clientButtons,
		us:                 us,
	}
}

func (b *Base) AcceptOfferHandler(c telebot.Context) error {
	err := b.us.UpdateSign(c.Sender().ID, true)
	if err != nil {
		return err
	}

	return c.Send(fmt.Sprintf("Добро пожаловать, %s!", c.Sender().FirstName), b.clientButtons.AddBtns())
}

func (b *Base) StartHandler(c telebot.Context) error {
	found, err := b.us.IsFound(c.Sender().ID)
	if err != nil {
		return err
	}

	if !found {
		var partnerID *int
		data := c.Data()
		if data != "" {
			parsedID, err := strconv.Atoi(data)
			if err != nil {
				partnerID = nil
			} else {
				partnerID = &parsedID
			}
		} else {
			partnerID = nil
		}

		user := models.User{
			ID:        c.Sender().ID,
			Username:  c.Sender().Username,
			Firstname: c.Sender().FirstName,
			Lastname:  c.Sender().LastName,
			PartnerID: partnerID,
			IsAdmin:   false,
		}

		err = b.us.Add(user)
		if err != nil {
			return err
		}
	}

	if sign, err := b.us.IsSign(c.Sender().ID); err == nil && sign {
		return c.Send(fmt.Sprintf("Добро пожаловать, %s!", c.Sender().FirstName), b.clientButtons.AddBtns())
	}
	return c.Send("Чтобы начать пользоваться NSVPN, необходимо принять условия публичной [оферты](https://teletype.in/@nsvpn/Dpvwcj7llQx).", b.acceptOfferButtons.AddBtns(), telebot.ModeMarkdown)
}

func (b *Base) HelpHandler(c telebot.Context) error {
	err := c.Send("🚀 Базовые команды\n/help - Посмотреть справку о командах\n")
	if err != nil {
		return err
	}
	return nil
}
