package services

import (
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/constants"
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

func (ps *Payments) GetAll(userID int64) (payments []*models.Payment, err error) {
	if userID == 0 {
		return nil, constants.ErrEmptyFields
	}

	return ps.pr.GetAll(userID)
}

func (ps *Payments) Get(userID int64, payload string) (payment *models.Payment, err error) {
	if userID == 0 || payload == "" {
		return nil, constants.ErrEmptyFields
	}

	return ps.pr.Get(userID, payload)
}

func (ps *Payments) Add(payment *models.Payment) error {
	if payment.UserID == 0 || payment.Amount == 0 || payment.Payload == "" || payment.Date.After(time.Now()) {
		return constants.ErrEmptyFields
	}

	return ps.pr.Add(payment)
}

func (ps *Payments) Update(userID int64, payload string, newPayment *models.Payment) error {
	if userID == 0 || payload == "" || newPayment == nil {
		return constants.ErrEmptyFields
	}

	return ps.pr.Update(userID, payload, newPayment)
}

func (ps *Payments) UpdateIsCompleted(userID int64, payload string, isCompleted bool) error {
	if userID == 0 || payload == "" {
		return constants.ErrEmptyFields
	}

	return ps.pr.UpdateIsCompleted(userID, payload, isCompleted)
}

func (ps *Payments) Delete(userID int64, payload string) error {
	if userID == 0 || payload == "" {
		return constants.ErrEmptyFields
	}

	return ps.pr.Delete(userID, payload)
}

func (ps *Payments) CreateTelegramInvoice(amount float64, title, description, payload string) telebot.Invoice {
	return telebot.Invoice{
		Title:       title,
		Description: description,
		Payload:     payload,
		Currency:    "XTR",
		Prices: []telebot.Price{
			{
				Label:  "XTR",
				Amount: int(amount),
			},
		},
	}
}
