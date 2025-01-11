package services

import (
	"fmt"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"time"
)

type Payments struct {
	pr *repository.Payments
}

func NewPayments(pr *repository.Payments) *Payments {
	return &Payments{
		pr: pr,
	}
}

func (p *Payments) CreateInvoice(currency string, uuid string, amount int, endDate time.Time) telebot.Invoice {
	now := time.Now()
	months := int(endDate.Sub(now).Hours() / (24 * 30))

	var description string
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

	invoice := telebot.Invoice{
		Title:       "Оплата подписки",
		Description: description,
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
	return p.pr.GetCurrencyID(currency)
}

func (p *Payments) Add(payment models.Payment) error {
	if payment.CurrencyID == 0 || payment.Amount == 0 || payment.SubscriptionID == 0 || payment.UserID == 0 || payment.Payload == "" {
		return constants.ErrUserNotFound
	}

	return p.pr.Add(payment)
}

func (p *Payments) UpdateStatus(userID int64, payload string, statusID int) error {
	if userID == 0 || payload == "" {
		return constants.ErrUserNotFound
	}

	return p.pr.UpdateStatus(userID, payload, statusID)
}
