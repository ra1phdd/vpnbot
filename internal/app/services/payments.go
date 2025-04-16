package services

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v4"
	"math"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
	"time"
)

type Payments struct {
	log *logger.Logger
	pr  *repository.Payments
}

func NewPayments(log *logger.Logger, pr *repository.Payments) *Payments {
	return &Payments{
		log: log,
		pr:  pr,
	}
}

func (ps *Payments) CreateInvoice(currency string, uuid string, amount int, endDate time.Time) telebot.Invoice {
	return telebot.Invoice{
		Title:       "Оплата подписки",
		Description: ps.GetDescriptionByDate(endDate),
		Payload:     uuid,
		Currency:    currency,
		Prices: []telebot.Price{
			{
				Label:  currency,
				Amount: amount,
			},
		},
	}
}

func (ps *Payments) GetDescriptionByDate(endDate time.Time) (description string) {
	months := math.Round(time.Until(endDate).Hours() / 24 / 30)

	switch months {
	case 1:
		description = "Подписка NSVPN на 1 месяц"
	case 2, 3, 4:
		description = fmt.Sprintf("Подписка NSVPN на %.0f месяца", months)
	case 12:
		description = "Подписка NSVPN на 1 год"
	default:
		description = fmt.Sprintf("Подписка NSVPN на %.0f месяцев", months)
	}
	return description
}

func (ps *Payments) GetAll(userId int64) ([]models.Payment, error) {
	if userId == 0 {
		return nil, errors.New("userId is empty")
	}

	return ps.pr.GetAll(userId)
}

func (ps *Payments) Get(userId int64, payload string) (models.Payment, error) {
	if userId == 0 || payload == "" {
		return models.Payment{}, errors.New("userId or payload is empty")
	}

	return ps.pr.Get(userId, payload)
}

func (ps *Payments) Add(payment models.Payment) error {
	if payment.UserID == 0 || payment.Amount == 0 || payment.CurrencyID == 0 || payment.SubscriptionID == 0 || payment.Payload == "" {
		return errors.New("userId, amount, currencyId, sebscriptionId or payload is empty")
	}
	if payment.Date.After(time.Now()) {
		return errors.New("date is invalid")
	}

	return ps.pr.Add(payment)
}

func (ps *Payments) Update(userId int64, payload string, payment models.Payment) error {
	if userId == 0 || payload == "" || payment == (models.Payment{}) {
		return errors.New("userId, payload or key is empty")
	}

	return ps.pr.Update(userId, payload, payment)
}

func (ps *Payments) UpdateIsCompleted(userId int64, payload string, isCompleted bool) error {
	if payload == "" || userId == 0 {
		return errors.New("user id or payload is empty")
	}

	return ps.pr.UpdateIsCompleted(userId, payload, isCompleted)
}

func (ps *Payments) Delete(userId int64, payload string) error {
	if userId == 0 || payload == "" {
		return errors.New("userId or payload is empty")
	}

	return ps.pr.Delete(userId, payload)
}
