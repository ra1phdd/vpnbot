package repository

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log/slog"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/logger"
	"time"
)

type Subscriptions struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *cache.Cache

	Plans  *SubscriptionsPlans
	Prices *SubscriptionsPrices
}

func NewSubscriptions(log *logger.Logger, db *gorm.DB, cache *cache.Cache) *Subscriptions {
	return &Subscriptions{
		log:   log,
		db:    db,
		cache: cache,
		Plans: &SubscriptionsPlans{
			log:   log,
			db:    db,
			cache: cache,
		},
		Prices: &SubscriptionsPrices{
			log:   log,
			db:    db,
			cache: cache,
		},
	}
}

func (sr *Subscriptions) GetAllActive() (subscriptions []*models.Subscription, err error) {
	cacheKey := "subscriptions:active"
	if err = sr.cache.Get(cacheKey, subscriptions); err == nil {
		sr.log.Debug("Returning user subscriptions from cache", slog.String("cache_key", cacheKey))
		return subscriptions, nil
	}

	if err = sr.db.Where("is_active = ?", true).Find(&subscriptions).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sr.log.Debug("No subscriptions found for user")
			return nil, nil
		}

		sr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	sr.cache.Set(cacheKey, subscriptions, 15*time.Minute)
	sr.log.Debug("Returning user subscriptions from DB", slog.String("cache_key", cacheKey))
	return subscriptions, nil
}

func (sr *Subscriptions) GetAllByUserID(userID int64) (subscriptions []*models.Subscription, err error) {
	cacheKey := fmt.Sprintf("subscriptions:user_id:%d:all", userID)
	if err = sr.cache.Get(cacheKey, subscriptions); err == nil {
		sr.log.Debug("Returning user subscriptions from cache", slog.String("cache_key", cacheKey), slog.Int64("userID", userID))
		return subscriptions, nil
	}

	if err = sr.db.Where("user_id = ?", userID).Find(&subscriptions).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sr.log.Debug("No subscriptions found for user", slog.Int64("userID", userID))
			return nil, nil
		}

		sr.log.Error("Failed to get data from db", err, slog.Int64("userID", userID))
		return nil, err
	}

	sr.cache.Set(cacheKey, subscriptions, 15*time.Minute)
	sr.log.Debug("Returning user subscriptions from DB", slog.String("cache_key", cacheKey), slog.Int64("userID", userID))
	return subscriptions, nil
}

func (sr *Subscriptions) GetLastByUserID(userID int64, isActive bool) (subscription *models.Subscription, err error) {
	cacheKey := fmt.Sprintf("subscription:user_id:%d:last", userID)
	if err = sr.cache.Get(cacheKey, subscription); err == nil {
		sr.log.Debug("Returning last subscription from cache", slog.String("cache_key", cacheKey), slog.Int64("userID", userID))
		return subscription, nil
	}

	if err = sr.db.Where("user_id = ? AND is_active = ?", userID, isActive).Order("id DESC").Limit(1).Find(&subscription).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sr.log.Debug("Subscription not found", slog.Int64("userID", userID))
			return nil, nil
		}

		sr.log.Error("Failed to get data from db", err, slog.Int64("userID", userID))
		return nil, err
	}

	sr.cache.Set(cacheKey, subscription, 15*time.Minute)
	sr.log.Debug("Returning last subscription from DB", slog.String("cache_key", cacheKey), slog.Int64("userID", userID))
	return subscription, nil
}

func (sr *Subscriptions) Add(subscription *models.Subscription) (int, error) {
	if err := sr.db.Create(&subscription).Error; err != nil {
		sr.log.Error("Failed to execute query from db", err, slog.Any("subscription", subscription))
		return 0, err
	}

	sr.cache.Delete(fmt.Sprintf("subscriptions:user_id:%d:all", subscription.UserID), fmt.Sprintf("subscription:user_id:%d:last", subscription.UserID))
	sr.log.Debug("Added new subscription", slog.Int("subId", subscription.ID))
	return subscription.ID, nil
}

func (sr *Subscriptions) UpdateEndDate(subID int, userID int64, endDate time.Time) error {
	if err := sr.db.Model(&models.Subscription{}).
		Where("id = ?", subID).
		Update("end_date", endDate).Error; err != nil {
		sr.log.Error("Failed to execute query from db", err, slog.Int("subID", subID))
		return err
	}

	sr.cache.Delete(fmt.Sprintf("subscriptions:user_id:%d:all", userID), fmt.Sprintf("subscription:user_id:%d:last", userID))
	sr.log.Debug("Updated subscription end date", slog.Int("subID", subID))
	return nil
}

func (sr *Subscriptions) UpdateIsActive(subID int, userID int64, isActive bool) error {
	if err := sr.db.Model(&models.Subscription{}).
		Where("id = ?", subID).
		Update("is_active", isActive).Error; err != nil {
		sr.log.Error("Failed to execute query from db", err, slog.Int("subID", subID))
		return err
	}

	sr.cache.Delete(fmt.Sprintf("subscriptions:user_id:%d:all", userID), fmt.Sprintf("subscription:user_id:%d:last", userID))
	sr.log.Debug("Updated subscription active status", slog.Int("subID", subID), slog.Bool("isActive", isActive))
	return nil
}

func (sr *Subscriptions) Delete(subID int, userID int64) error {
	if err := sr.db.Where("id = ?", subID).Delete(&models.Subscription{}).Error; err != nil {
		sr.log.Error("Failed to execute query from db", err, slog.Int("subID", subID))
		return err
	}

	sr.cache.Delete(fmt.Sprintf("subscriptions:user_id:%d:all", userID), fmt.Sprintf("subscription:user_id:%d:last", userID))
	sr.log.Debug("Deleted subscription", slog.Int("subID", subID))
	return nil
}
