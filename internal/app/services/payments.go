package services

import (
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
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

	return p.pr.GetCurrencyID(currency)
}

func (p *Payments) Add(payment models.Payment) error {
	if payment.CurrencyID == 0 || payment.Amount == 0 || payment.SubscriptionID == 0 || payment.UserID == 0 || payment.Payload == "" {
		logger.Error("One of the payment fields is empty", zap.Int("currencyID", payment.CurrencyID), zap.Float64("amount", payment.Amount), zap.Int("subscriptionID", payment.SubscriptionID), zap.Int64("userID", payment.UserID), zap.String("payload", payment.Payload))
		return fmt.Errorf("one of the payment fields is empty")
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
