package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/db"
	"strconv"
	"time"
)

type Payments struct{}

func NewPayments() *Payments {
	return &Payments{}
}

func (p *Payments) GetAllByUserId(userId int64) (payments []models.Payment, err error) {
	cacheKey := fmt.Sprintf("payment:user_id:%d", userId)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &payments)
		if err != nil {
			return nil, err
		}
		return payments, nil
	}

	rows, err := db.Conn.Queryx(`SELECT * FROM payments WHERE user_id = $1`, userId)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var data models.Payment
		if err := rows.StructScan(&data); err != nil {
			return nil, err
		}

		payments = append(payments, data)
	}
	if len(payments) == 0 {
		return nil, constants.ErrPromoCodeNotFound
	}

	jsonData, err := json.Marshal(payments)
	if err != nil {
		return nil, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 0).Err()
	if err != nil {
		return nil, err
	}

	return payments, nil
}

func (p *Payments) Get(userId int64, payload string) (payment models.Payment, err error) {
	cacheKey := fmt.Sprintf("payment:user_id:%d:payload:%s", userId, payload)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return models.Payment{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &payment)
		if err != nil {
			return models.Payment{}, err
		}
		return payment, nil
	}

	err = db.Conn.QueryRowx(`SELECT * FROM payments WHERE user_id = $1 AND payload = $2`, userId, payload).StructScan(&payment)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Payment{}, constants.ErrServerNotFound
		}
		return models.Payment{}, err
	}

	jsonData, err := json.Marshal(payment)
	if err != nil {
		return models.Payment{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 0).Err()
	if err != nil {
		return models.Payment{}, err
	}

	return payment, nil
}

func (p *Payments) Add(data models.Payment) error {
	_, err := db.Conn.Exec(`INSERT INTO payments (user_id, amount, currency_id, subscription_id, payload, is_completed) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, data.UserID, data.Amount, data.CurrencyID, data.SubscriptionID, data.Payload, data.IsCompleted)
	return err
}

func (p *Payments) Update(payment models.Payment) error {
	paymentOld, err := p.Get(payment.UserID, payment.Payload)
	if err != nil {
		return err
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if payment.Amount != paymentOld.Amount && payment.Amount > 0 {
		_, err = tx.Exec(`UPDATE payments SET amount = $1 WHERE id = $2`, payment.Amount, paymentOld.ID)
		if err != nil {
			return err
		}
	}

	if payment.CurrencyID != paymentOld.CurrencyID && payment.CurrencyID != 0 {
		_, err = tx.Exec(`UPDATE payments SET currency_id = $1 WHERE id = $2`, payment.CurrencyID, paymentOld.ID)
		if err != nil {
			return err
		}
	}

	if payment.Date != paymentOld.Date && !payment.Date.After(time.Now()) {
		_, err = tx.Exec(`UPDATE payments SET date = $1 WHERE id = $2`, payment.Date, paymentOld.ID)
		if err != nil {
			return err
		}
	}

	if payment.SubscriptionID != paymentOld.SubscriptionID && payment.SubscriptionID != 0 {
		_, err = tx.Exec(`UPDATE payments SET subscription_id = $1 WHERE id = $2`, payment.SubscriptionID, paymentOld.ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *Payments) UpdateIsCompleted(userId int64, payload string, isCompeleted bool) error {
	_, err := db.Conn.Exec(`UPDATE payments SET is_completed = $1 WHERE user_id = $2 AND payload = $3`, isCompeleted, userId, payload)
	return err
}

func (p *Payments) GetCurrencyID(currency string) (int, error) {
	var currencyID int
	cacheKey := fmt.Sprintf("currency:%s", currency)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, err
	} else if cacheValue != "" {
		return strconv.Atoi(cacheValue)
	}

	err = db.Conn.QueryRowx(`SELECT id FROM currencies WHERE currency_code = $1`, currency).Scan(&currencyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, constants.ErrServerNotFound
		}
		return 0, err
	}

	err = cache.Rdb.Set(cache.Ctx, cacheKey, currencyID, 0).Err()
	if err != nil {
		return 0, err
	}

	return currencyID, nil
}
