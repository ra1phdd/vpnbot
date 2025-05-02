package handlers

import (
	"fmt"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"strconv"
	"strings"
	"time"
)

type Payments struct {
	log                  *logger.Logger
	bot                  *telebot.Bot
	cs                   *services.Currency
	ps                   *services.Payments
	ss                   *services.Subscriptions
	us                   *services.Users
	clientButtonsWithSub *services.Buttons
}

func NewPayments(log *logger.Logger, bot *telebot.Bot, ps *services.Payments, cs *services.Currency, ss *services.Subscriptions, us *services.Users, clientButtonsWithSub *services.Buttons) *Payments {
	return &Payments{
		log:                  log,
		bot:                  bot,
		cs:                   cs,
		ps:                   ps,
		ss:                   ss,
		us:                   us,
		clientButtonsWithSub: clientButtonsWithSub,
	}
}

func (p *Payments) ChooseCurrencyHandler(c telebot.Context, amount float64, payload string) error {
	currencies, err := p.cs.GetAll()
	if err != nil {
		return err
	}

	buttons, layouts := p.cs.ProcessButtons(currencies)
	curBtns := services.NewButtons(buttons, layouts, "inline")

	for _, cur := range currencies {
		p.bot.Handle(curBtns.GetBtns()["currency_"+strings.ToLower(cur.Code)], func(c telebot.Context) error {
			if amount == 0 {
				return p.requestAmount(c, cur, payload)
			}

			return p.PaymentHandler(c, amount, payload, cur)
		})
	}

	return c.Send("Выберите валюту:", curBtns.AddBtns())
}

func (p *Payments) requestAmount(c telebot.Context, currency *models.Currency, payload string) error {
	_, err := c.Bot().Send(c.Recipient(), fmt.Sprintf("Введите сумму в %s:", currency.Code))
	if err != nil {
		return err
	}

	p.bot.Handle(telebot.OnText, func(c telebot.Context) error {
		amount, err := strconv.ParseFloat(c.Text(), 64)
		if err != nil {
			return c.Send("Некорректная сумма, попробуйте ещё раз")
		}

		p.bot.Handle(telebot.OnText, nil)

		return p.PaymentHandler(c, amount, payload, currency)
	})

	return nil
}

func (p *Payments) PaymentHandler(c telebot.Context, amount float64, payload string, currency *models.Currency) error {
	payment := &models.Payment{
		UserID:      c.Sender().ID,
		Amount:      amount / currency.ExchangeRate,
		Date:        time.Now(),
		Payload:     payload,
		IsCompleted: false,
	}

	err := p.ps.Add(payment)
	if err != nil {
		p.log.Error("Failed add payment", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	var invoice telebot.Invoice
	switch currency.Code {
	case "XTR":
		invoice = p.ps.CreateTelegramInvoice(amount, "Оплата", fmt.Sprintf("Пополнение баланса на %0.f %s", amount, currency.Code), payload)
	default:
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}
	return c.Send(&invoice)
}

func (p *Payments) PreCheckoutHandler(c telebot.Context) error {
	err := p.ps.UpdateIsCompleted(c.Sender().ID, c.PreCheckoutQuery().Payload, true)
	if err != nil {
		p.log.Error("Failed update isCompleted", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	currency, err := p.cs.Get("XTR")
	if err != nil {
		return err
	}
	amount := int(float64(c.PreCheckoutQuery().Total) / currency.ExchangeRate)

	err = p.us.IncrementBalance(c.Sender().ID, amount)
	if err != nil {
		p.log.Error("Failed update isCompleted", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
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
	//	return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	//}

	return c.Send("Платёж успешно завершен! Номер платежа: "+c.PreCheckoutQuery().Payload, p.clientButtonsWithSub.AddBtns())
}
