package handlers

import (
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"math"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"strconv"
)

type Payments struct {
	log *logger.Logger
	bot *telebot.Bot
	cs  *services.Currency
	ps  *services.Payments
	us  *services.Users
}

func NewPayments(log *logger.Logger, bot *telebot.Bot, ps *services.Payments, cs *services.Currency, us *services.Users) *Payments {
	return &Payments{
		log: log,
		bot: bot,
		cs:  cs,
		ps:  ps,
		us:  us,
	}
}

func (p *Payments) RequestAmount(c telebot.Context, payload, note string) error {
	btns := getReplyButtons(c)
	baseCurrency, err := p.cs.GetIsBase()
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	err = c.EditOrSend(fmt.Sprintf("💳 Введите сумму в %s:", baseCurrency.Code))
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	p.bot.Handle(telebot.OnText, func(c telebot.Context) error {
		amount, err := strconv.ParseFloat(c.Text(), 64)
		if err != nil || amount < 1 {
			return c.Send("❌ Некорректная сумма, попробуйте ещё раз", btns)
		}

		p.bot.Handle(telebot.OnText, nil)
		return p.ChooseCurrencyHandler(c, math.Round(amount), payload, note, false)
	})

	return nil
}

func (p *Payments) ChooseCurrencyHandler(c telebot.Context, amount float64, payload, note string, isBuySub bool) error {
	btns := getReplyButtons(c)
	baseCurrency, err := p.cs.GetIsBase()
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	user, err := p.us.Get(c.Sender().ID)
	if err != nil {
		return err
	}

	chooseBtns := services.NewButtons([]models.ButtonOption{
		{
			Value:   "pay_bankcard",
			Display: "💳 Банковская карта/СБП",
		},
		{
			Value:   "pay_stars",
			Display: "⭐ Telegram Stars",
		},
		{
			Value:   "pay_cryptocurrency",
			Display: "💎 Криптовалюта",
		},
	}, []int{1, 1, 1}, "inline")

	msg := fmt.Sprintf("💵 Сумма к оплате: %.f %s\n📦 Номер платежа: %s\n\nВыберите удобный для Вас способ оплаты:", amount, baseCurrency.Code, payload)
	if user.Balance > 0 && isBuySub {
		amount -= user.Balance
		msg = fmt.Sprintf("💰 Ваш текущий баланс: %.f %s\n💵 Сумма к оплате: %.f %s\n📦 Номер платежа: %s\n\nВыберите удобный для Вас способ оплаты:", user.Balance, baseCurrency.Code, amount, baseCurrency.Code, payload)
	}

	p.bot.Handle(chooseBtns.GetBtn("pay_bankcard"), func(c telebot.Context) error {
		return p.PaymentHandler(c, amount, "Оплата подписки на NSVPN", payload, note, "RUB")
	})
	p.bot.Handle(chooseBtns.GetBtn("pay_stars"), func(c telebot.Context) error {
		return p.PaymentHandler(c, amount, "Оплата подписки на NSVPN", payload, note, "XTR")
	})
	p.bot.Handle(chooseBtns.GetBtn("pay_cryptocurrency"), func(c telebot.Context) error {
		//return p.PaymentHandler(c, amount, "Оплата подписки на NSVPN", payload, note, "USD")
		return c.Send("Не реализовано")
	})

	return c.Send(msg, chooseBtns.AddBtns())
}

func (p *Payments) PaymentHandler(c telebot.Context, amount float64, description, payload, note, currencyCode string) error {
	btns := getReplyButtons(c)
	currency, err := p.cs.Get(currencyCode)
	if err != nil {
		return err
	}

	payment := &models.Payment{
		UserID:      c.Sender().ID,
		Amount:      amount,
		Type:        "income",
		Payload:     payload,
		Note:        note,
		IsCompleted: false,
	}

	err = p.ps.Add(payment)
	if err != nil {
		p.log.Error("Failed add payment", err)
		return c.Send(constants.UserError, btns)
	}

	p.bot.Handle(telebot.OnCheckout, func(c telebot.Context) error {
		return p.PreCheckoutHandler(c, amount)
	})
	invoice := p.ps.CreateInvoice(math.Round(amount*currency.ExchangeRate), "Оплата", description, currency.Code, "", payload)
	return c.Send(&invoice)
}

func (p *Payments) PreCheckoutHandler(c telebot.Context, amount float64) error {
	btns := getReplyButtons(c)
	err := p.ps.UpdateIsCompleted(c.Sender().ID, c.PreCheckoutQuery().Payload, true)
	if err != nil {
		p.log.Error("Failed update isCompleted", err)
		return c.Send(constants.UserError, btns)
	}

	err = p.us.IncrementBalance(c.Sender().ID, amount)
	if err != nil {
		p.log.Error("Failed update isCompleted", err)
		return c.Send(constants.UserError, btns)
	}

	//err = p.bot.Accept(c.PreCheckoutQuery())
	//if err != nil {
	//	err := p.ps.UpdateIsCompleted(c.Sender().ID, c.PreCheckoutQuery().Payload, false)
	//	if err != nil {
	//		p.log.Error("Failed update isCompleted", err)
	//	}
	//
	//	err = p.us.DecrementBalance(c.Sender().ID, amount)
	//	if err != nil {
	//		p.log.Error("Failed update isCompleted", err)
	//	}
	//
	//	p.log.Error("Failed update isCompleted", err)
	//	return c.Send(constants.UserError, btns)
	//}

	user, err := p.us.Get(c.Sender().ID)
	if err != nil {
		p.log.Error("Failed update isCompleted", err)
	} else if user.PartnerID != 0 && amount*0.15 >= 1 {
		err = p.ps.Add(&models.Payment{
			UserID:      user.PartnerID,
			Amount:      amount * 0.15,
			Type:        "income",
			Payload:     uuid.New().String(),
			Note:        "15% от пополнения баланса рефералом",
			IsCompleted: true,
		})
		if err != nil {
			p.log.Error("Failed add payment", err)
		}

		err = p.us.IncrementBalance(user.PartnerID, math.Round(amount*0.15))
		if err != nil {
			p.log.Error("Failed increment balance", err)
		}
	}

	return c.Send("✅ Платёж успешно завершен!", btns)
}

func (p *Payments) HistoryPaymentsHandler(c telebot.Context, currentPage int) error {
	const pageSize = 15
	btns := getReplyButtons(c)

	totalCount, err := p.ps.GetPaymentsCount(c.Sender().ID)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}
	if totalCount == 0 {
		return c.Send("🧾 У вас пока нету оплаченных платежей", btns)
	}

	totalPages := totalCount / pageSize
	if totalCount%pageSize != 0 {
		totalPages++
	}

	if currentPage < 1 || currentPage > int(totalPages) {
		return nil
	}

	offset := (currentPage - 1) * pageSize
	payments, err := p.ps.GetAll(c.Sender().ID, offset, pageSize)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	baseCurrency, err := p.cs.GetIsBase()
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	msg := fmt.Sprintf("🧾 История платежей (страница %d из %d):\n", currentPage, totalPages)
	for i, payment := range payments {
		msg += fmt.Sprintf("%d) %s - %s - %.f %s\n", i+1+offset, payment.Date.Format("2006-01-02 15:04:05"), payment.Note, payment.Amount, baseCurrency.Code)
	}

	pgBtns := services.NewButtons([]models.ButtonOption{
		{
			Value:   "pagination_first",
			Display: "⏪",
		},
		{
			Value:   "pagination_prev",
			Display: "◀",
		},
		{
			Value:   "pagination_next",
			Display: "▶",
		},
		{
			Value:   "pagination_last",
			Display: "⏩",
		},
	}, []int{4}, "inline")
	p.bot.Handle(pgBtns.GetBtn("pagination_first"), func(c telebot.Context) error {
		return p.HistoryPaymentsHandler(c, 1)
	})
	p.bot.Handle(pgBtns.GetBtn("pagination_prev"), func(c telebot.Context) error {
		return p.HistoryPaymentsHandler(c, currentPage-1)
	})
	p.bot.Handle(pgBtns.GetBtn("pagination_next"), func(c telebot.Context) error {
		return p.HistoryPaymentsHandler(c, currentPage+1)
	})
	p.bot.Handle(pgBtns.GetBtn("pagination_last"), func(c telebot.Context) error {
		return p.HistoryPaymentsHandler(c, int(totalPages))
	})

	btns = &telebot.ReplyMarkup{}
	if totalPages > 1 {
		btns = pgBtns.AddBtns()
	}
	return c.EditOrSend(msg, btns)
}
