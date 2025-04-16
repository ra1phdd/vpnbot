package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"log/slog"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/logger"
	"time"
)

type Subscriptions struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *redis.Client
}

func NewSubscriptions(log *logger.Logger, db *gorm.DB, cache *redis.Client) *Subscriptions {
	return &Subscriptions{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (sr *Subscriptions) GetAllByUserId(userId int64) ([]models.Subscription, error) {
	cacheKey := fmt.Sprintf("subscriptions:%d:all", userId)
	cacheValue, err := sr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		sr.log.Error(constants.ErrGetDataFromCache, err, slog.Int64("userId", userId))
		return nil, err
	}

	var subs []models.Subscription
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &subs); err != nil {
			sr.log.Error(constants.ErrUnmarshalDataFromJSON, err, slog.Int64("userId", userId))
			return nil, err
		}
		return subs, nil
	}

	if err := sr.db.Where("user_id = ?", userId).Find(&subs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		sr.log.Error(constants.ErrGetDataFromDB, err, slog.Int64("userId", userId))
		return nil, err
	}

	jsonData, err := json.Marshal(subs)
	if err != nil {
		sr.log.Error(constants.ErrMarshalDataToJSON, err, slog.Int64("userId", userId))
		return nil, err
	}

	if err := sr.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		sr.log.Error(constants.ErrSetDataToCache, err, slog.Int64("userId", userId))
		return nil, err
	}

	return subs, nil
}

func (sr *Subscriptions) GetLastByUserId(userId int64, isActive bool) (models.Subscription, error) {
	cacheKey := fmt.Sprintf("subscription:%d", userId)
	cacheValue, err := sr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		sr.log.Error(constants.ErrGetDataFromCache, err, slog.Int64("userId", userId))
		return models.Subscription{}, err
	}

	var sub models.Subscription
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &sub); err != nil {
			sr.log.Error(constants.ErrUnmarshalDataFromJSON, err, slog.Int64("userId", userId))
			return models.Subscription{}, err
		}
		return sub, nil
	}

	if err := sr.db.Where("user_id = ? AND is_active = ?", userId, isActive).Order("id DESC").Limit(1).Find(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Subscription{}, nil
		}

		sr.log.Error(constants.ErrGetDataFromDB, err, slog.Int64("userId", userId))
		return models.Subscription{}, err
	}

	jsonData, err := json.Marshal(sub)
	if err != nil {
		sr.log.Error(constants.ErrMarshalDataToJSON, err, slog.Int64("userId", userId))
		return models.Subscription{}, err
	}

	if err := sr.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		sr.log.Error(constants.ErrSetDataToCache, err, slog.Int64("userId", userId))
		return models.Subscription{}, err
	}

	return sub, nil
}

func (sr *Subscriptions) Add(data models.Subscription) (int, error) {
	if err := sr.db.Create(&data).Error; err != nil {
		sr.log.Error(constants.ErrExecQueryFromDB, err, slog.Any("subscription", data))
		return 0, err
	}
	return data.ID, nil
}

func (sr *Subscriptions) UpdateEndDate(subId int, userId int64, endDate time.Time) error {
	if err := sr.db.Model(&models.Subscription{}).
		Where("id = ?", subId).
		Update("end_date", endDate).Error; err != nil {
		sr.log.Error(constants.ErrExecQueryFromDB, err, slog.Int("subId", subId))
		return err
	}

	cacheKeys := []string{
		fmt.Sprintf("subscription:%d", userId),
		fmt.Sprintf("subscription:%d:all", userId),
	}
	for _, key := range cacheKeys {
		if err := sr.cache.Del(context.Background(), key).Err(); err != nil {
			sr.log.Error(constants.ErrDeleteDataFromCache, err)
			return err
		}
	}
	return nil
}

func (sr *Subscriptions) UpdateIsActive(subId int, userId int64, isActive bool) error {
	if err := sr.db.Model(&models.Subscription{}).
		Where("id = ?", subId).
		Update("is_active", isActive).Error; err != nil {
		sr.log.Error(constants.ErrExecQueryFromDB, err, slog.Int("subId", subId))
		return err
	}

	cacheKeys := []string{
		fmt.Sprintf("subscription:%d", userId),
		fmt.Sprintf("subscription:%d:all", userId),
	}
	for _, key := range cacheKeys {
		if err := sr.cache.Del(context.Background(), key).Err(); err != nil {
			sr.log.Error(constants.ErrDeleteDataFromCache, err)
			return err
		}
	}
	return nil
}

func (sr *Subscriptions) Delete(subId int, userId int64) error {
	if err := sr.db.Where("id = ?", subId).Delete(&models.Subscription{}).Error; err != nil {
		sr.log.Error(constants.ErrExecQueryFromDB, err, slog.Int("subId", subId))
		return err
	}

	cacheKeys := []string{
		fmt.Sprintf("subscription:%d", userId),
		fmt.Sprintf("subscription:%d:all", userId),
	}
	for _, key := range cacheKeys {
		if err := sr.cache.Del(context.Background(), key).Err(); err != nil {
			sr.log.Error(constants.ErrDeleteDataFromCache, err)
			return err
		}
	}
	return nil
}
