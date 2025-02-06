package handlers

import (
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
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

	u, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	sub := models.Subscription{
		UserID:   c.Sender().ID,
		EndDate:  &endDate,
		IsActive: false,
	}

	subId, err := p.ss.Add(sub)
	if err != nil {
		logger.Error("Failed add subscription", zap.Error(err))
		return err
	}

	currencyID, err := p.ps.ConvertCurrencyToId("XTR")
	if err != nil {
		logger.Error("Failed convert currency to id", zap.Error(err))
		return err
	}

	payment := models.Payment{
		UserID:         c.Sender().ID,
		Amount:         float64(amount),
		CurrencyID:     currencyID,
		Date:           time.Now(),
		SubscriptionID: subId,
		Payload:        fmt.Sprint(u),
		IsCompleted:    false,
	}

	err = p.ps.Add(payment)
	if err != nil {
		logger.Error("Failed add payment", zap.Error(err))
		return err
	}

	invoice := p.ps.CreateInvoice("XTR", fmt.Sprint(u), amount, endDate)
	return c.Send(&invoice)
}

func (p *Payments) PreCheckoutHandler(c telebot.Context) error {
	err := p.ps.UpdateIsCompleted(c.Sender().ID, c.PreCheckoutQuery().Payload, true)
	if err != nil {
		logger.Error("Failed update isCompleted", zap.Error(err))
		return err
	}

	err = p.ss.UpdateIsActive(c.Sender().ID, c.PreCheckoutQuery().Payload, true)
	if err != nil {
		logger.Error("Failed update isActive", zap.Error(err))
		return err
	}

	//err = e.Bot.Accept(c.PreCheckoutQuery())
	//if err != nil {
	//	return err
	//}

	return c.Send(fmt.Sprintf("Платёж успешно завершен! Номер платежа: %s", c.PreCheckoutQuery().Payload))
}
