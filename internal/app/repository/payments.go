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

func (p *Payments) Add(data models.Payment) (int, error) {
	var id int
	err := db.Conn.QueryRowx(`INSERT INTO payments (user_id, amount, currency, subscription_id, uuid) VALUES ($1, $2, $3, $4, $5) RETURNING id`, data.UserID, data.Amount, data.CurrencyID, data.SubscriptionID, data.Payload).Scan(&id)
	return id, err
}
