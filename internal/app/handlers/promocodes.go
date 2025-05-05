package handlers

import (
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/internal/app/state"
	"nsvpn/pkg/logger"
	"strconv"
	"time"
)

type Promocodes struct {
	log    *logger.Logger
	bot    *telebot.Bot
	ps     *services.Payments
	pcodes *services.Promocodes
	us     *services.Users

	skipBtn *services.Buttons
}

func NewPromocodes(log *logger.Logger, bot *telebot.Bot, ps *services.Payments, pcodes *services.Promocodes, us *services.Users) *Promocodes {
	return &Promocodes{
		log:    log,
		bot:    bot,
		ps:     ps,
		pcodes: pcodes,
		us:     us,

		skipBtn: services.NewButtons([]models.ButtonOption{
			{
				Value:   "skip_promocode",
				Display: "Пропустить",
			},
		}, []int{1}, "inline"),
	}
}

func (p *Promocodes) RequestPromocodeHandler(c telebot.Context, ps state.Storage[state.PaymentsState]) error {
	btns := getReplyButtons(c)
	err := c.Send("💳 Введите промокод:", p.skipBtn.AddBtns())
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	err = c.Respond()
	if err != nil {
		p.log.Error("Failed to send message", err)
	}

	resultChan := make(chan string)
	p.bot.Handle(p.skipBtn.GetBtn("skip_promocode"), func(c telebot.Context) error {
		defer func(c telebot.Context) {
			err := c.Respond()
			if err != nil {
				p.log.Error("Failed to send message", err)
			}
		}(c)

		resultChan <- ""
		return nil
	})

	p.bot.Handle(telebot.OnText, func(c telebot.Context) error {
		resultChan <- c.Text()

		p.bot.Handle(telebot.OnText, func(c telebot.Context) error {
			return c.Send("🤔 Неизвестная команда. Используйте /help для получения списка команд", btns)
		})
		return nil
	})

	promocode := <-resultChan
	if promocode == "" {
		return nil
	}

	return p.GetPromocodeHandler(c, promocode, ps)
}

func (p *Promocodes) GetPromocodeHandler(c telebot.Context, code string, ps state.Storage[state.PaymentsState]) error {
	btns := getReplyButtons(c)
	promocode, err := p.pcodes.Get(code)
	if err != nil {
		return c.Send(constants.UserHasNoRights, btns)
	}

	count, err := p.ps.GetPaymentsCount(c.Sender().ID)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	switch {
	case promocode == nil || !promocode.IsActive:
		return c.Send("❌ Промокод не найден", btns)
	case promocode.TotalActivations != 0 && promocode.CurrentActivations >= promocode.TotalActivations:
		return c.Send("❌ Количество активаций промокода превышено", btns)
	case promocode.OnlyNewUsers && count > 0:
		return c.Send("❌ Промокод доступен только для новых пользователей", btns)
	case promocode.EndAt != nil && promocode.EndAt.Before(time.Now()):
		return c.Send("❌ Срок действия промокода истёк", btns)
	}

	activation, err := p.pcodes.Activations.Get(promocode.ID, c.Sender().ID)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}
	if activation != nil && activation.ID != 0 {
		return c.Send("❌ Промокод уже был применён", btns)
	}

	ps.Update(strconv.FormatInt(c.Sender().ID, 10), func(ps state.PaymentsState) state.PaymentsState {
		ps.Amount -= ps.Amount * float64(promocode.Discount) / 100
		ps.Promocode = promocode
		return ps
	})

	return c.Send("✅ Промокод применён", btns)
}
