package handlers

import (
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"time"
)

type Promocodes struct {
	log    *logger.Logger
	bot    *telebot.Bot
	ps     *services.Payments
	pcodes *services.Promocodes
	us     *services.Users
}

func NewPromocodes(log *logger.Logger, bot *telebot.Bot, ps *services.Payments, pcodes *services.Promocodes, us *services.Users) *Promocodes {
	return &Promocodes{
		log:    log,
		bot:    bot,
		ps:     ps,
		pcodes: pcodes,
		us:     us,
	}
}

func (p *Promocodes) RequestPromocodeHandler(c telebot.Context) (uint, int, error) {
	btns := getReplyButtons(c)

	skipBtn := services.NewButtons([]models.ButtonOption{
		{
			Value:   "skip_promocode",
			Display: "Пропустить",
		},
	}, []int{1}, "inline")
	err := c.Send("💳 Введите промокод:", skipBtn.AddBtns())
	if err != nil {
		return 0, 0, c.Send(constants.UserError, btns)
	}

	resultChan := make(chan string)
	p.bot.Handle(skipBtn.GetBtn("skip_promocode"), func(_ telebot.Context) error {
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
		return 0, 0, nil
	}

	return p.GetPromocodeHandler(c, promocode)
}

func (p *Promocodes) GetPromocodeHandler(c telebot.Context, code string) (uint, int, error) {
	btns := getReplyButtons(c)
	promocode, err := p.pcodes.Get(code)
	if err != nil {
		return 0, 0, c.Send(constants.UserHasNoRights, btns)
	}

	count, err := p.ps.GetPaymentsCount(c.Sender().ID)
	if err != nil {
		return 0, 0, c.Send(constants.UserError, btns)
	}

	switch {
	case promocode == nil || !promocode.IsActive:
		return 0, 0, c.Send("❌ Промокод не найден", btns)
	case promocode.TotalActivations != 0 && promocode.CurrentActivations >= promocode.TotalActivations:
		return 0, 0, c.Send("❌ Количество активаций промокода превышено", btns)
	case promocode.OnlyNewUsers && count > 0:
		return 0, 0, c.Send("❌ Промокод доступен только для новых пользователей", btns)
	case promocode.EndAt != nil && promocode.EndAt.Before(time.Now()):
		return 0, 0, c.Send("❌ Срок действия промокода истёк", btns)
	}

	activation, err := p.pcodes.Activations.Get(promocode.ID, c.Sender().ID)
	if err != nil {
		return 0, 0, c.Send(constants.UserError, btns)
	}
	if activation != nil && activation.ID != 0 {
		return 0, 0, c.Send("❌ Промокод уже был применён", btns)
	}

	return promocode.ID, promocode.Discount, c.Send("✅ Промокод применён", btns)
}
