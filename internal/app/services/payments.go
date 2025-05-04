package services

import (
	"encoding/json"
	"fmt"
	"gopkg.in/telebot.v4"
	"math"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
	"time"
)

type Payments struct {
	log *logger.Logger
	pr  *repository.Payments
	cr  *repository.Currency
}

func NewPayments(log *logger.Logger, pr *repository.Payments, cr *repository.Currency) *Payments {
	return &Payments{
		log: log,
		pr:  pr,
		cr:  cr,
	}
}

func (ps *Payments) GetAll(userID int64, offset, limit int) (payments []*models.Payment, err error) {
	if userID == 0 {
		return nil, constants.ErrEmptyFields
	}

	return ps.pr.GetAll(userID, offset, limit)
}

func (ps *Payments) GetPaymentsCount(userID int64) (int64, error) {
	if userID == 0 {
		return 0, constants.ErrEmptyFields
	}

	return ps.pr.GetPaymentsCount(userID)
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

type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type Item struct {
	Description    string `json:"description"`
	Quantity       int    `json:"quantity"`
	Amount         Amount `json:"amount"`
	VatCode        int    `json:"vat_code"`
	Measure        string `json:"measure"`
	PaymentMode    string `json:"payment_mode"`
	PaymentSubject string `json:"payment_subject"`
}

type Receipt struct {
	Items []Item `json:"items"`
}

type PaymentData struct {
	Receipt Receipt `json:"receipt"`
}

func (ps *Payments) CreateInvoice(amount float64, title, description, email, currency, providerToken, payload string) telebot.Invoice {
	invoice := telebot.Invoice{
		Title:       title,
		Description: description,
		Payload:     payload,
		Currency:    currency,
		Token:       providerToken,
		Prices: []telebot.Price{
			{
				Label:  "К оплате",
				Amount: int(math.Round(amount)),
			},
		},
	}

	if currency == "RUB" {
		receipt := PaymentData{
			Receipt{
				Items: []Item{
					{
						Description: description,
						Amount: Amount{
							Value:    fmt.Sprintf("%.2f", amount),
							Currency: currency,
						},
						VatCode:        2,
						Quantity:       1,
						Measure:        "piece",
						PaymentSubject: "payment",
						PaymentMode:    "full_payment",
					},
				},
			},
		}

		jsonData, err := json.Marshal(receipt)
		if err != nil {
			ps.log.Error("Failed marshal to json", err)
		}

		invoice.Prices[0].Amount *= 100
		invoice.NeedEmail = true
		invoice.SendEmail = true
		invoice.Data = string(jsonData)
	}

	return invoice
}
