package handlers

import (
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"time"
)

type Users struct {
	log *logger.Logger
	bot *telebot.Bot
	us  *services.Users
	ss  *Subscriptions
	ph  *Payments
}

func NewUsers(log *logger.Logger, bot *telebot.Bot, us *services.Users, ss *Subscriptions, ph *Payments) *Users {
	return &Users{
		log: log,
		bot: bot,
		us:  us,
		ss:  ss,
		ph:  ph,
	}
}

func (u *Users) ProfileHandler(c telebot.Context) error {
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
	u.bot.Handle(balanceBtns.GetBtn("top_balance"), func(c telebot.Context) error {
		return u.ph.RequestAmount(c, uuid.New().String(), "Пополнение баланса")
	})
	if subOk && sub != nil && sub.EndDate.After(time.Now().UTC()) && sub.IsActive {
		u.bot.Handle(balanceBtns.GetBtn("extend_sub"), u.ss.ChooseDurationHandler)
	}
	u.bot.Handle(balanceBtns.GetBtn("history_payments"), func(c telebot.Context) error {
		return u.ph.HistoryPaymentsHandler(c, 1, true)
	})

	partners, err := u.us.CountPartners(c.Sender().ID)
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
