package services

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gopkg.in/telebot.v4"
	"math"
	"net/http"
	"nsvpn/internal/app/config"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
	"time"
)

type Payments struct {
	log *logger.Logger
	cfg *config.Configuration
	pr  *repository.Payments
	cr  *repository.Currency
}

func NewPayments(log *logger.Logger, cfg *config.Configuration, pr *repository.Payments, cr *repository.Currency) *Payments {
	return &Payments{
		log: log,
		cfg: cfg,
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

func (ps *Payments) CreateInvoice(amount float64, title, description, payload string) telebot.Invoice {
	invoice := telebot.Invoice{
		Title:       title,
		Description: description,
		Payload:     payload,
		Currency:    "XTR",
		Prices: []telebot.Price{
			{
				Label:  "К оплате",
				Amount: int(math.Round(amount)),
			},
		},
	}

	return invoice
}

func (ps *Payments) CreateBankcardPayment(amount float64, email, description string) models.YoukassaRequest {
	return models.YoukassaRequest{
		Amount: models.YoukassaAmount{
			Value:    fmt.Sprintf("%.2f", amount),
			Currency: "RUB",
		},
		Capture: true,
		Confirmation: models.YoukassaConfirmation{
			Type:      "redirect",
			ReturnURL: "https://t.me/nsvpn_bot",
		},
		Description: description,
		Receipt: models.YoukassaReceipt{
			Customer: models.YoukassaCustomer{
				Email: email,
			},
			Items: []models.YoukassaItem{
				{
					Description: description,
					Amount: models.YoukassaAmount{
						Value:    fmt.Sprintf("%.2f", amount),
						Currency: "RUB",
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
}

func (ps *Payments) CheckBankcardPayment(id string) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", ps.cfg.YoukassaURL, id), nil)
	if err != nil {
		ps.log.Error("Error creating request", err)
		return err
	}
	req.SetBasicAuth(ps.cfg.YoukassaID, ps.cfg.YoukassaAPI)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	client := &http.Client{Timeout: 30 * time.Second}
	for {
		select {
		case <-ctx.Done():
			return constants.ErrPaymentTimeExpired
		case <-ticker.C:
			resp, err := client.Do(req)
			if err != nil {
				ps.log.Error("Error making request", err)
				continue
			}

			var response models.YoukassaResponse
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				ps.log.Error("Error decoding payment response", err)
				return err
			}

			if response.Status == "succeeded" {
				return nil
			} else if response.Status == "canceled" {
				return constants.ErrCancelPayment
			}
		}
	}
}

func (ps *Payments) CreateCryptoPayment(amount float64, description, payload string) models.HeleketRequest {
	return models.HeleketRequest{
		Amount:                 fmt.Sprintf("%.2f", amount),
		Currency:               "RUB",
		OrderID:                payload,
		ReturnURL:              "https://t.me/nsvpn_bot",
		SuccessURL:             "https://t.me/nsvpn_bot",
		Lifetime:               600,
		Subtract:               50,
		AccuracyPaymentPercent: 2,
		AdditionalData:         description,
	}
}

func (ps *Payments) CheckCryptoPayment(uuid, orderID string) error {
	jsonData, err := json.Marshal(models.HeleketInfoRequest{
		UUID:    uuid,
		OrderID: orderID,
	})
	if err != nil {
		ps.log.Error("Error marshaling JSON", err)
		return err
	}

	b64 := base64.StdEncoding.EncodeToString(jsonData)
	hash := md5.Sum([]byte(b64 + ps.cfg.HeleketAPI))

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/info", ps.cfg.HeleketURL), nil)
	if err != nil {
		ps.log.Error("Error creating request", err)
		return err
	}
	req.Header.Set("merchant", ps.cfg.HeleketID)
	req.Header.Set("sign", hex.EncodeToString(hash[:]))
	req.Header.Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	client := &http.Client{Timeout: 30 * time.Second}
	for {
		select {
		case <-ctx.Done():
			return constants.ErrPaymentTimeExpired
		case <-ticker.C:
			resp, err := client.Do(req)
			if err != nil {
				ps.log.Error("Error making request", err)
				continue
			}

			var response models.HeleketResponse
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				ps.log.Error("Error decoding payment response", err)
				return err
			}

			if response.Result.PaymentStatus == "paid" || response.Result.PaymentStatus == "paid_over" {
				return nil
			} else if response.Result.PaymentStatus == "cancel" {
				return constants.ErrCancelPayment
			}
		}
	}
}
