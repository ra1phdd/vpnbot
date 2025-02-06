package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/db"
	"time"
)

type Subscriptions struct{}

func NewSubscriptions() *Subscriptions {
	return &Subscriptions{}
}

func (s *Subscriptions) GetLastByUserId(userId int64) (sub models.Subscription, err error) {
	cacheKey := fmt.Sprintf("subscription:%d", userId)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return models.Subscription{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &sub)
		return sub, err
	}

	err = db.Conn.QueryRowx(`SELECT * FROM subscriptions WHERE user_id = $1 ORDER BY id DESC LIMIT 1`, userId).StructScan(&sub)
	if err != nil {
		return models.Subscription{}, err
	}

	jsonData, err := json.Marshal(sub)
	if err != nil {
		return models.Subscription{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		return models.Subscription{}, err
	}

	return sub, nil
}

func (s *Subscriptions) Add(data models.Subscription) (id int, err error) {
	err = db.Conn.QueryRowx(`INSERT INTO subscriptions (user_id, end_date, is_active) VALUES ($1, $2, $3) RETURNING id`, data.UserID, data.EndDate, data.IsActive).Scan(&id)
	return id, err
}

func (s *Subscriptions) UpdateEndDate(userId int, endDate time.Time) error {
	_, err := db.Conn.Exec(`UPDATE subscriptions SET end_date = $1 WHERE user_id = $2`, endDate, userId)
	return err
}

func (s *Subscriptions) UpdateIsActive(subId int, isActive bool) error {
	_, err := db.Conn.Exec(`UPDATE subscriptions SET is_active = $1 WHERE id = $2`, isActive, subId)
	return err
}
