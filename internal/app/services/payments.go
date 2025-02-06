package services

import (
	"fmt"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"time"
)

type Payments struct {
	pr *repository.Payments
	cr *repository.Currency
}

func NewPayments(pr *repository.Payments, cr *repository.Currency) *Payments {
	return &Payments{
		pr: pr,
		cr: cr,
	}
}

func (p *Payments) CreateInvoice(currency string, uuid string, amount int, endDate time.Time) telebot.Invoice {
	invoice := telebot.Invoice{
		Title:       "Оплата подписки",
		Description: p.GetDescriptionByDate(endDate),
		Payload:     uuid,
		Currency:    currency,
		Prices: []telebot.Price{
			{
				Label:  currency,
				Amount: amount,
			},
		},
	}

	return invoice
}

func (p *Payments) ConvertCurrencyToId(currency string) (int, error) {
	if currency == "" {
		return 0, fmt.Errorf("currency is empty")
	}

	cur, err := p.cr.Get(currency)
	if err != nil {
		return 0, err
	}

	return cur.ID, nil
}

func (p *Payments) Add(payment models.Payment) error {
	switch {
	case payment.UserID == 0:
		return fmt.Errorf("userId is empty")
	case payment.Amount == 0:
		return fmt.Errorf("amount is empty")
	case payment.CurrencyID == 0:
		return fmt.Errorf("currencyId is empty")
	case payment.Date.After(time.Now()):
		return fmt.Errorf("date is invalid")
	case payment.SubscriptionID == 0:
		return fmt.Errorf("subscriptionID is invalid")
	case payment.Payload == "":
		return fmt.Errorf("payload is invalid")
	}

	return p.pr.Add(payment)
}

func (p *Payments) UpdateIsCompleted(userId int64, payload string, isCompleted bool) error {
	if payload == "" || userId == 0 {
		return fmt.Errorf("user id or payload is empty")
	}

	return p.pr.UpdateIsCompleted(userId, payload, isCompleted)
}

func (p *Payments) GetDescriptionByDate(endDate time.Time) (description string) {
	now := time.Now()
	months := int(endDate.Sub(now).Hours() / (24 * 30))

	switch months {
	case 1:
		description = "Подписка NSVPN на 1 месяц."
	case 2, 3, 4:
		description = fmt.Sprintf("Подписка NSVPN на %d месяца.", months)
	case 12:
		description = "Подписка NSVPN на 1 год."
	default:
		description = fmt.Sprintf("Подписка NSVPN на %d месяцев.", months)
	}
	return description
}
