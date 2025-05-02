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

type SubscriptionsPrices struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *cache.Cache
}

func (sr *SubscriptionsPrices) GetAll() (price []*models.SubscriptionPrice, err error) {
	cacheKey := "subscription_price:all"
	if err = sr.cache.Get(cacheKey, price); err == nil {
		sr.log.Debug("Returning subscription price from cache", slog.String("cache_key", cacheKey))
		return price, nil
	}

	if err = sr.db.Find(&price).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sr.log.Debug("No subscription price found in database")
			return nil, nil
		}

		sr.log.Error("Failed to get price from db", err)
		return nil, err
	}

	sr.cache.Set(cacheKey, price, 1*time.Hour)
	sr.log.Debug("Returning subscription price from db", slog.String("cache_key", cacheKey))
	return price, nil
}

func (sr *SubscriptionsPrices) Get(id int) (price *models.SubscriptionPrice, err error) {
	cacheKey := fmt.Sprintf("subscription_price:%d", id)
	if err = sr.cache.Get(cacheKey, price); err == nil {
		sr.log.Debug("Returning subscription price from cache", slog.String("cache_key", cacheKey))
		return price, nil
	}

	if err = sr.db.Where("id = ?", id).First(&price).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sr.log.Debug("No subscription price found in database", slog.Int("id", id))
			return nil, nil
		}

		sr.log.Error("Failed to get price from db", err, slog.Int("id", id))
		return nil, err
	}

	sr.cache.Set(cacheKey, price, 1*time.Hour)
	sr.log.Debug("Returning subscription price from db", slog.String("cache_key", cacheKey))
	return price, nil
}

func (sr *SubscriptionsPrices) Add(price *models.SubscriptionPrice) error {
	if err := sr.db.Create(&price).Error; err != nil {
		sr.log.Error("Failed to create price in db", err, slog.Any("price", price))
		return err
	}

	sr.cache.Delete(fmt.Sprintf("subscription_price:%d", price.ID), "subscription_price:all")
	sr.log.Debug("Added new subscription price in db", slog.Int("id", price.ID))
	return nil
}

func (sr *SubscriptionsPrices) UpdatePrice(id int, price float64) error {
	if err := sr.db.Model(&models.Subscription{}).
		Where("id = ?", id).
		Update("price", price).Error; err != nil {
		sr.log.Error("Failed to execute query from db", err, slog.Int("id", id))
		return err
	}

	sr.cache.Delete(fmt.Sprintf("subscription_price:%d", id), "subscription_price:all")
	sr.log.Debug("Updated subscription price", slog.Int("id", id))
	return nil
}

func (sr *SubscriptionsPrices) Delete(id int) error {
	if err := sr.db.Delete(&models.SubscriptionPrice{}, id).Error; err != nil {
		sr.log.Error("Failed to delete price from db", err, slog.Int("id", id))
		return err
	}

	sr.cache.Delete(fmt.Sprintf("subscription_price:%d", id), "subscription_price:all")
	sr.log.Debug("Deleted subscription price", slog.Int("id", id))
	return nil
}
