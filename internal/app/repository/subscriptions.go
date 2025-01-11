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
	"time"
)

type Subscriptions struct{}

func NewSubscriptions() *Subscriptions {
	return &Subscriptions{}
}

func (s *Subscriptions) GetByUserId(userId int64) (models.Subscription, error) {
	var data models.Subscription

	cacheKey := fmt.Sprintf("subscription:%d", userId)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return models.Subscription{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &data)
		if err != nil {
			return models.Subscription{}, err
		}
	}

	err = db.Conn.QueryRowx(`SELECT * FROM subscriptions WHERE user_id = $1 ORDER BY id DESC LIMIT 1`, userId).StructScan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Subscription{}, constants.ErrServerNotFound
		}
		return models.Subscription{}, err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return models.Subscription{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 0).Err()
	if err != nil {
		return models.Subscription{}, err
	}

	return data, nil
}

func (s *Subscriptions) Add(data models.Subscription) (int, error) {
	var id int
	err := db.Conn.QueryRowx(`INSERT INTO subscriptions (user_id, end_date) VALUES ($1, $2) RETURNING id`, data.UserID, data.EndDate).Scan(&id)
	return id, err
}

func (s *Subscriptions) UpdateEndDate(userId int, endDate time.Time) error {
	_, err := db.Conn.Exec(`UPDATE subscriptions SET end_date = $1 WHERE user_id = $2`, endDate, userId)
	return err
}
