package handlers

import (
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"log/slog"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"time"
)

type Base struct {
	log *logger.Logger
	bot *telebot.Bot
	us  *services.Users
	ss  *Subscriptions
	ph  *Payments
}

func NewBase(log *logger.Logger, bot *telebot.Bot, us *services.Users, ss *Subscriptions, ph *Payments) *Base {
	return &Base{
		log: log,
		bot: bot,
		us:  us,
		ss:  ss,
		ph:  ph,
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

func (b *Base) HelpHandler(c telebot.Context) error {
	btns := getReplyButtons(c)
	return c.Send("🚀 Базовые команды\n/help - Посмотреть справку о командах\n", btns)
}

func (b *Base) OnTextHandler(c telebot.Context) error {
	btns := getReplyButtons(c)
	return c.Send("🤔 Неизвестная команда. Используйте /help для получения списка команд", btns)
}

func (b *Base) ProfileHandler(c telebot.Context) error {
	btns := getReplyButtons(c)
	user, userOk := c.Get("user").(*models.User)
	if !userOk {
		return c.Send(constants.UserError, btns)
	}
	sub, subOk := c.Get("sub").(*models.Subscription)

	subMsg := "🎟️ *Статус подписки*: неактивно ❌"
	btnOpts := []models.ButtonOption{
		{
			Value:   "top_balance",
			Display: "💸 Пополнить баланс",
		},
	}
	layout := []int{1}
	if subOk && sub != nil && sub.EndDate.After(time.Now().UTC()) && sub.IsActive {
		subMsg = "🎟️ *Статус подписки*: активно ✅\n📅 *Срок окончания*: " + sub.EndDate.Format("02-01-2006 15:04:05")
		btnOpts = append(btnOpts, models.ButtonOption{
			Value:   "extend_sub",
			Display: "⏳ Продлить подписку",
		})
		layout = append(layout, 1)
	}
	btnOpts = append(btnOpts, models.ButtonOption{
		Value:   "history_payments",
		Display: "🧾 История платежей",
	})
	layout = append(layout, 1)

	balanceBtns := services.NewButtons(btnOpts, layout, "inline")
	for _, btn := range balanceBtns.GetBtns() {
		switch btn.Unique {
		case "top_balance":
			b.bot.Handle(btn, func(c telebot.Context) error {
				return b.ph.RequestAmount(c, uuid.New().String(), "Пополнение баланса")
			})
		case "extend_sub":
			b.bot.Handle(btn, b.ss.ChooseDurationHandler)
		case "history_payments":
			b.bot.Handle(btn, func(c telebot.Context) error {
				return b.ph.HistoryPaymentsHandler(c, 1)
			})
		}
	}

	partners, err := b.us.CountPartners(c.Sender().ID)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	return c.Send(
		fmt.Sprintf("👔 *Ваш профиль*:\n\n🙎🏻 *Имя:* %s\n🆔 *ID:* %d\n\n💰 *Баланс*: %0.f₽\n🤝 *Кол-во рефералов*: %d чел.\n\n%s", c.Sender().FirstName, c.Sender().ID, user.Balance, partners, subMsg),
		&telebot.SendOptions{
			ReplyMarkup: balanceBtns.AddBtns(),
			ParseMode:   telebot.ModeMarkdown,
		},
	)
}

func (b *Base) InfoHandler(c telebot.Context) error {
	btns := getReplyButtons(c)
	return c.Send("💡 Информация", btns)
}
