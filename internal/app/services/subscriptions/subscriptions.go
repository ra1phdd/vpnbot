package subscriptions

import (
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

func New() *Subscriptions {
	return &Subscriptions{}
}

func (s *Subscriptions) IsActive(userId int64) (bool, error) {
	data, err := s.GetByUserId(userId)
	if err != nil {
		return false, err
	}

	if data.EndDate.Before(time.Now().UTC()) {
		return true, nil
	}
	return false, nil
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

	rows, err := db.Conn.Query(`SELECT * FROM subscriptions WHERE user_id = $1 ORDER BY id DESC LIMIT 1`, userId)
	if err != nil {
		return models.Subscription{}, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&data.ID, &data.UserID, &data.StartDate, &data.EndDate)
		if err != nil {
			return models.Subscription{}, err
		}
	}
	if data.ID == 0 {
		return models.Subscription{}, constants.ErrSubNotFound
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
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *Subscriptions) UpdateEndDate(userId int, endDate time.Time) error {
	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE subscriptions SET end_date = $1 WHERE user_id = $2`, endDate, userId)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}
