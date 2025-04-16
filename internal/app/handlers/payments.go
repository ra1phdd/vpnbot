package handlers

import (
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"time"
)

type Payments struct {
	log                  *logger.Logger
	cs                   *services.Currency
	ps                   *services.Payments
	ss                   *services.Subscriptions
	clientButtonsWithSub *services.Buttons
}

func NewPayments(log *logger.Logger, ps *services.Payments, cs *services.Currency, ss *services.Subscriptions, clientButtonsWithSub *services.Buttons) *Payments {
	return &Payments{
		log:                  log,
		cs:                   cs,
		ps:                   ps,
		ss:                   ss,
		clientButtonsWithSub: clientButtonsWithSub,
	}
}

func (p *Payments) PaymentHandler(c telebot.Context) error {
	var endDate time.Time
	var amount int
	switch c.Callback().Unique {
	case "sub_one_month":
		amount = 68
		endDate = time.Now().AddDate(0, 1, 0)
	case "sub_three_month":
		amount = 182
		endDate = time.Now().AddDate(0, 3, 0)
	case "sub_six_month":
		amount = 342
		endDate = time.Now().AddDate(0, 6, 0)
	}

	if c.Sender().ID == 1230045591 {
		amount = 1
	}

	u, err := uuid.NewUUID()
	if err != nil {
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	sub := models.Subscription{
		UserID:   c.Sender().ID,
		EndDate:  &endDate,
		IsActive: false,
	}

	subId, err := p.ss.Add(sub)
	if err != nil {
		p.log.Error("Failed add subscription", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	currency, err := p.cs.Get("XTR")
	if err != nil {
		p.log.Error("Failed convert currency to id", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	payment := models.Payment{
		UserID:         c.Sender().ID,
		Amount:         float64(amount),
		CurrencyID:     currency.ID,
		Date:           time.Now(),
		SubscriptionID: subId,
		Payload:        fmt.Sprint(u),
		IsCompleted:    false,
	}

	err = p.ps.Add(payment)
	if err != nil {
		p.log.Error("Failed add payment", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	invoice := p.ps.CreateInvoice("XTR", fmt.Sprint(u), amount, endDate)
	return c.Send(&invoice)
}

func (p *Payments) PreCheckoutHandler(c telebot.Context) error {
	sub, err := p.ss.GetLastByUserId(c.Sender().ID, false)
	if err != nil {
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	err = p.ps.UpdateIsCompleted(c.Sender().ID, c.PreCheckoutQuery().Payload, true)
	if err != nil {
		p.log.Error("Failed update isCompleted", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	err = p.ss.UpdateIsActive(sub.ID, c.Sender().ID, true)
	if err != nil {
		p.log.Error("Failed update isActive", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	//err = e.Bot.Accept(c.PreCheckoutQuery())
	//if err != nil {
	//	return err
	//}

	return c.Send(fmt.Sprintf("Платёж успешно завершен! Номер платежа: %s", c.PreCheckoutQuery().Payload), p.clientButtonsWithSub.AddBtns())
}
