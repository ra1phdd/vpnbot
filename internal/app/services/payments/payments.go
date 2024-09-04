package payments

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/db"
)

type Payments interface {
}

type Service struct {
	Payments Payments
}

func New() *Service {
	return &Service{}
}

func (s Service) GetByUserId(subscriptionId int64) (models.Payment, error) {
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

	rows, err := db.Conn.Query(`SELECT * FROM payments WHERE subscription_id = $1 ORDER BY id DESC LIMIT 1`, subscriptionId)
	if err != nil {
		return models.Payment{}, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&data.ID, &data.UserID, &data.Amount, &data.Currency, &data.Date, &data.SubscriptionID, &data.Uuid)
		if err != nil {
			return models.Payment{}, err
		}
	}
	if data.ID == 0 {
		return models.Payment{}, constants.ErrPaymentNotFound
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

func (s Service) Add(data models.Payment) (int, error) {
	var id int
	err := db.Conn.QueryRowx(`INSERT INTO payments (user_id, amount, currency, subscription_id, uuid) VALUES ($1, $2, $3, $4, $5) RETURNING id`, data.UserID, data.Amount, data.Currency, data.SubscriptionID, data.Uuid).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}
