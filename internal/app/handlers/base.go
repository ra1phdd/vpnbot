package handlers

import (
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"log/slog"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"strconv"
	"time"
)

type Base struct {
	log                  *logger.Logger
	bot                  *telebot.Bot
	clientButtons        *services.Buttons
	clientButtonsWithSub *services.Buttons
	us                   *services.Users
	ss                   *services.Subscriptions
	ph                   *Payments
}

func NewBase(log *logger.Logger, bot *telebot.Bot, clientButtons, clientButtonsWithSub *services.Buttons, us *services.Users, ss *services.Subscriptions, ph *Payments) *Base {
	return &Base{
		log:                  log,
		bot:                  bot,
		clientButtons:        clientButtons,
		clientButtonsWithSub: clientButtonsWithSub,
		us:                   us,
		ss:                   ss,
		ph:                   ph,
	}
}

func (b *Base) AcceptOfferHandler(c telebot.Context) error {
	err := b.us.UpdateIsSign(c.Sender().ID, true)
	if err != nil {
		b.log.Error("Failed to update sign", err, slog.Int64("userId", c.Sender().ID))
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	return c.Send(fmt.Sprintf("Добро пожаловать, %s!", c.Sender().FirstName), b.clientButtons.AddBtns())
}

func (b *Base) StartHandler(c telebot.Context) error {
	data, err := b.us.Get(c.Sender().ID)
	if err != nil {
		b.log.Error("Failed to fetch user", err, slog.Int64("userId", c.Sender().ID))
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	if data == nil {
		var partnerID int64
		if c.Data() != "" {
			parsedID, err := strconv.ParseInt(c.Data(), 10, 64)
			if err == nil {
				partnerID = parsedID
			}
		}

		data = &models.User{
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

	acceptOfferButtons := services.NewButtons(models.AcceptOfferButton, []int{1}, "inline")
	b.bot.Handle(acceptOfferButtons.GetBtns()["accept_offer"], b.AcceptOfferHandler)
	return c.Send("Чтобы начать пользоваться NSVPN, необходимо принять условия публичной [оферты](https://teletype.in/@nsvpn/Dpvwcj7llQx).", acceptOfferButtons.AddBtns(), telebot.ModeMarkdown)
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

func (b *Base) ProfileHandler(c telebot.Context) error {
	sub, err := b.ss.GetLastByUserID(c.Sender().ID, true)
	if err != nil {
		return err
	}

	subMsg := "🎟️ *Статус подписки*: неактивно ❌"
	if sub != nil && sub.EndDate.After(time.Now().UTC()) && sub.IsActive {
		subMsg = "🎟️ *Статус подписки*: активно ✅\n📅 *Срок окончания*: " + sub.EndDate.Format("02-01-2006 15:04:05")
	}

	user, err := b.us.Get(c.Sender().ID)
	if err != nil {
		return err
	}

	btns := services.NewButtons([]models.ButtonOption{{Value: "top_balance", Display: "Пополнить баланс"}}, []int{1}, "inline")
	for _, btn := range btns.GetBtns() {
		b.bot.Handle(btn, func(c telebot.Context) error {
			u, err := uuid.NewUUID()
			if err != nil {
				return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
			}

			return b.ph.ChooseCurrencyHandler(c, 0, u.String())
		})
	}

	reffers := 0
	return c.Send(
		fmt.Sprintf("📝 *Ваш профиль*:\n\n🙎🏻 *Имя:* %s\n🆔 *ID:* %d\n\n💰 *Баланс*: %d₽\n🤝 *Кол-во рефералов*: %d чел.\n\n%s", c.Sender().FirstName, c.Sender().ID, user.Balance, reffers, subMsg),
		&telebot.SendOptions{
			ReplyMarkup: btns.AddBtns(),
			ParseMode:   telebot.ModeMarkdown,
		},
	)
}
