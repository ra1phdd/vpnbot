package handlers

import (
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v3"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"time"
)

type Payments struct {
	ps *services.Payments
	ss *services.Subscriptions
}

func NewPayments(ps *services.Payments, ss *services.Subscriptions) *Payments {
	return &Payments{
		ps: ps,
		ss: ss,
	}
}

func (p *Payments) PaymentHandler(c telebot.Context) error {
	isActive, err := p.ss.IsActive(c.Sender().ID)
	if err != nil {
		return err
	}

	if isActive {
		return c.Send("Ошибка! У вас уже есть действующая подписка. Дождитесь её окончания")
	}

	u, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	invoice := CreateInvoice("XTR", fmt.Sprint(u))

	err = c.Send(&invoice)
	if err != nil {
		return err
	}

	return nil
}

func (p *Payments) PreCheckoutHandler(c telebot.Context) error {
	sub := models.Subscription{
		UserID:  c.Sender().ID,
		EndDate: time.Now().UTC().AddDate(0, 0, 30),
	}

	subId, err := p.ss.Add(sub)
	if err != nil {
		return err
	}

	payment := models.Payment{
		UserID:         c.Sender().ID,
		Amount:         c.PreCheckoutQuery().Total,
		Currency:       c.PreCheckoutQuery().Currency,
		SubscriptionID: subId,
		Uuid:           c.PreCheckoutQuery().Payload,
	}

	_, err = p.ps.Add(payment)
	if err != nil {
		return err
	}

	//err = e.Bot.Accept(c.PreCheckoutQuery())
	//if err != nil {
	//	return err
	//}

	return c.Send(fmt.Sprintf("Платёж успешно завершен! Номер платежа: %s", c.PreCheckoutQuery().Payload))
}

func CreateInvoice(currency string, uuid string) telebot.Invoice {
	var amount int
	switch currency {
	case "XTR":
		amount = 100
	default:
		amount = 1
	}

	invoice := telebot.Invoice{
		Title:       "Оплата подписки",
		Description: "Подписка NSVPN на 1 месяц.",
		Payload:     uuid,
		Currency:    currency,
		Prices: []telebot.Price{
			{
				Label:  "Оплата подписки",
				Amount: amount,
			},
		},
	}

	return invoice
}
