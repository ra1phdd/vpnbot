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
)

type Payments struct{}

func NewPayments() *Payments {
	return &Payments{}
}

func (p *Payments) GetByUserId(subscriptionId int64) (models.Payment, error) {
	var data models.Payment

	cacheKey := fmt.Sprintf("payment:%d", subscriptionId)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return models.Payment{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &data)
		if err != nil {
			return models.Payment{}, err
		}
	}

	err = db.Conn.QueryRowx(`SELECT * FROM payments WHERE subscription_id = $1 ORDER BY id DESC LIMIT 1`, subscriptionId).StructScan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Payment{}, constants.ErrServerNotFound
		}
		return models.Payment{}, err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return models.Payment{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 0).Err()
	if err != nil {
		return models.Payment{}, err
	}

	return data, nil
}

func (p *Payments) Add(data models.Payment) error {
	_, err := db.Conn.Exec(`INSERT INTO payments (user_id, amount, currency_id, subscription_id, payload, status_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, data.UserID, data.Amount, data.CurrencyID, data.SubscriptionID, data.Payload, data.StatusID)
	return err
}

func (p *Payments) UpdateStatus(userID int64, payload string, statusID int) error {
	result, err := db.Conn.Exec(`UPDATE payments SET status_id = $1 WHERE user_id = $2 AND payload = $3`, statusID, userID, payload)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows were updated for user_id=%d and payload=%s", userID, payload)
	}

	return nil
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
