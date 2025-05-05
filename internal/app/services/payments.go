package services

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gopkg.in/telebot.v4"
	"log/slog"
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
}

func NewPayments(log *logger.Logger, cfg *config.Configuration, pr *repository.Payments) *Payments {
	return &Payments{
		log: log,
		cfg: cfg,
		pr:  pr,
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

func (ps *Payments) CreateBankcardPayment(amount float64, email, description, payload string) (*models.YoukassaResponse, error) {
	request := models.YoukassaRequest{
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

	jsonData, err := json.Marshal(request)
	if err != nil {
		ps.log.Error("Error marshaling JSON", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, ps.cfg.YoukassaURL, bytes.NewBuffer(jsonData))
	if err != nil {
		ps.log.Error("Error creating request", err)
		return nil, err
	}

	req.SetBasicAuth(ps.cfg.YoukassaID, ps.cfg.YoukassaAPI)
	req.Header.Set("Idempotence-Key", payload)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		ps.log.Error("Error making request", err)
		return nil, err
	}
	if resp != nil && resp.Body != nil {
		defer func() {
			if closeErr := resp.Body.Close(); closeErr != nil {
				ps.log.Debug("Error closing response body", slog.Any("error", closeErr), slog.String("url", req.URL.String()))
			}
		}()
	}

	var response models.YoukassaResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		ps.log.Error("Error decoding payment response", err)
		return nil, err
	}

	return &response, nil
}

func (ps *Payments) CheckBankcardPayment(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s", ps.cfg.YoukassaURL, id), nil)
	if err != nil {
		ps.log.Error("Error creating request", err)
		return err
	}
	req.SetBasicAuth(ps.cfg.YoukassaID, ps.cfg.YoukassaAPI)

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
			if resp != nil && resp.Body != nil {
				defer func() {
					if closeErr := resp.Body.Close(); closeErr != nil {
						ps.log.Debug("Error closing response body", slog.Any("error", closeErr), slog.String("url", req.URL.String()))
					}
				}()
			}

			var response models.YoukassaResponse
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				ps.log.Error("Error decoding payment response", err)
				return err
			}

			switch response.Status {
			case "succeeded":
				return nil
			case "canceled":
				return constants.ErrCancelPayment
			}
		}
	}
}

func (ps *Payments) CreateCryptoPayment(amount float64, description, payload string) (*models.HeleketResponse, error) {
	request := models.HeleketRequest{
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

	jsonData, err := json.Marshal(request)
	if err != nil {
		ps.log.Error("Error marshaling JSON", err)
		return nil, err
	}

	b64 := base64.StdEncoding.EncodeToString(jsonData)
	hash := md5.Sum([]byte(b64 + ps.cfg.HeleketAPI))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, ps.cfg.HeleketURL, bytes.NewBuffer(jsonData))
	if err != nil {
		ps.log.Error("Error creating request", err)
		return nil, err
	}
	req.Header.Set("Merchant", ps.cfg.HeleketID)
	req.Header.Set("Sign", hex.EncodeToString(hash[:]))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		ps.log.Error("Error making request", err)
		return nil, err
	}
	if resp != nil && resp.Body != nil {
		defer func() {
			if closeErr := resp.Body.Close(); closeErr != nil {
				ps.log.Debug("Error closing response body", slog.Any("error", closeErr), slog.String("url", req.URL.String()))
			}
		}()
	}

	var response models.HeleketResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		ps.log.Error("Error decoding payment response", err)
		return nil, err
	}

	return &response, nil
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ps.cfg.HeleketURL+"/info", nil)
	if err != nil {
		ps.log.Error("Error creating request", err)
		return err
	}
	req.Header.Set("Merchant", ps.cfg.HeleketID)
	req.Header.Set("Sign", hex.EncodeToString(hash[:]))
	req.Header.Set("Content-Type", "application/json")

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

			switch response.Result.PaymentStatus {
			case "paid", "paid_over":
				return nil
			case "cancel":
				return constants.ErrCancelPayment
			}
		}
	}
}

func (ps *Payments) CreatePaymentMessage(amount float64, paymentTime time.Time, method, payload string) string {
	msg := "🧾 Счет на оплату подписки успешно создан.\n\n"
	msg += fmt.Sprintf("💸 Стоимость: %.0f руб.\n", amount)
	msg += fmt.Sprintf("💳 Метод оплаты: %s\n", method)
	msg += fmt.Sprintf("📦 Номер платежа: %s\n\n", payload)
	msg += fmt.Sprintf("Вам нужно оплатить счет до %s. При возникновении каких-либо проблем не стесняйтесь обращаться в нашу поддержку", paymentTime.Format("02-01-2006 15:04:05"))

	return msg
}
