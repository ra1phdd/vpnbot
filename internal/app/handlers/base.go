package handlers

import (
	"fmt"
	"gopkg.in/telebot.v4"
	"log/slog"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"strconv"
)

type Base struct {
	log                  *logger.Logger
	acceptOfferButtons   *services.Buttons
	clientButtons        *services.Buttons
	clientButtonsWithSub *services.Buttons
	us                   *services.Users
	ss                   *services.Subscriptions
	sh                   *Servers
}

func NewBase(log *logger.Logger, acceptOfferButtons, clientButtons, clientButtonsWithSub *services.Buttons, us *services.Users, ss *services.Subscriptions, sh *Servers) *Base {
	return &Base{
		log:                  log,
		acceptOfferButtons:   acceptOfferButtons,
		clientButtons:        clientButtons,
		clientButtonsWithSub: clientButtonsWithSub,
		us:                   us,
		ss:                   ss,
		sh:                   sh,
	}
}

func (b *Base) AcceptOfferHandler(c telebot.Context) error {
	err := b.us.Update(c.Sender().ID, models.User{IsSign: true})
	if err != nil {
		b.log.Error("Failed to update sign", err, slog.Int64("userId", c.Sender().ID))
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	return c.Send(fmt.Sprintf("Добро пожаловать, %s!", c.Sender().FirstName), b.clientButtons.AddBtns())
}

func (b *Base) StartHandler(c telebot.Context) error {
	data, err := b.us.GetById(c.Sender().ID)
	if err != nil {
		b.log.Error("Failed to fetch user", err, slog.Int64("userId", c.Sender().ID))
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	if data == (models.User{}) {
		var partnerID *int = nil
		if c.Data() != "" {
			parsedID, err := strconv.Atoi(c.Data())
			if err == nil {
				partnerID = &parsedID
			}
		}

		data = models.User{
			ID:        c.Sender().ID,
			Username:  c.Sender().Username,
			Firstname: c.Sender().FirstName,
			Lastname:  c.Sender().LastName,
			PartnerID: partnerID,
			IsAdmin:   false,
			IsSign:    false,
		}

		err = b.us.Add(data)
		if err != nil {
			b.log.Error("Failed to create new user", err, slog.Int64("userId", c.Sender().ID))
			return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
		}
	}

	if data.IsSign {
		if isActive, err := b.ss.IsActive(c.Sender().ID, true); err == nil && isActive {
			return c.Send(fmt.Sprintf("Добро пожаловать, %s!", c.Sender().FirstName), b.clientButtonsWithSub.AddBtns())
		}
		return c.Send(fmt.Sprintf("Добро пожаловать, %s!", c.Sender().FirstName), b.clientButtons.AddBtns())
	}
	return c.Send("Чтобы начать пользоваться NSVPN, необходимо принять условия публичной [оферты](https://teletype.in/@nsvpn/Dpvwcj7llQx).", b.acceptOfferButtons.AddBtns(), telebot.ModeMarkdown)
}

func (b *Base) HelpHandler(c telebot.Context) error {
	return c.Send("🚀 Базовые команды\n/help - Посмотреть справку о командах\n")
}

func (b *Base) OnTextHandler(c telebot.Context) error {
	if isActive, err := b.ss.IsActive(c.Sender().ID, true); err == nil && isActive {
		return c.Send("Неизвестная команда. Используйте /help для получения списка команд", b.clientButtonsWithSub.AddBtns())
	}
	return c.Send("Неизвестная команда. Используйте /help для получения списка команд", b.clientButtons.AddBtns())
}
