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

type Currency struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *redis.Client
}

func NewCurrency(log *logger.Logger, db *gorm.DB, cache *redis.Client) *Currency {
	return &Currency{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (cr *Currency) GetAll() ([]models.Currency, error) {
	cacheKey := "currency:all"
	cacheValue, err := cr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		cr.log.Error(constants.ErrGetDataFromCache, err)
		return nil, err
	}

	var currencies []models.Currency
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &currencies); err != nil {
			cr.log.Error(constants.ErrUnmarshalDataFromJSON, err)
			return nil, err
		}
		return currencies, nil
	}

	if err := cr.db.Find(&currencies).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		cr.log.Error(constants.ErrGetDataFromDB, err)
		return nil, err
	}

	jsonData, err := json.Marshal(currencies)
	if err != nil {
		cr.log.Error(constants.ErrMarshalDataToJSON, err)
		return nil, err
	}

	if err := cr.cache.Set(context.Background(), cacheKey, jsonData, 0).Err(); err != nil {
		cr.log.Error(constants.ErrSetDataToCache, err)
		return nil, err
	}

	return currencies, nil
}

func (cr *Currency) Get(currencyCode string) (models.Currency, error) {
	cacheKey := fmt.Sprintf("currency:%s", currencyCode)
	cacheValue, err := cr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		cr.log.Error(constants.ErrGetDataFromCache, err, slog.String("currencyCode", currencyCode))
		return models.Currency{}, err
	}

	var currency models.Currency
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &currency); err != nil {
			cr.log.Error(constants.ErrUnmarshalDataFromJSON, err, slog.String("currencyCode", currencyCode))
			return models.Currency{}, err
		}
		return currency, nil
	}

	if err := cr.db.Where("currency_code = ?", currencyCode).First(&currency).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Currency{}, nil
		}

		cr.log.Error(constants.ErrGetDataFromDB, err, slog.String("currencyCode", currencyCode))
		return models.Currency{}, err
	}

	jsonData, err := json.Marshal(currency)
	if err != nil {
		cr.log.Error(constants.ErrMarshalDataToJSON, err, slog.String("currencyCode", currencyCode))
		return models.Currency{}, err
	}
	if err := cr.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		cr.log.Error(constants.ErrSetDataToCache, err, slog.String("currencyCode", currencyCode))
		return models.Currency{}, err
	}

	return currency, nil
}

func (cr *Currency) Add(currency models.Currency) (int, error) {
	if err := cr.db.Create(&currency).Error; err != nil {
		cr.log.Error(constants.ErrExecQueryFromDB, err, slog.Any("currency", currency))
		return 0, err
	}
	return currency.ID, nil
}

func (cr *Currency) Delete(currencyCode string) error {
	if err := cr.db.Where("currency_code = ?", currencyCode).Delete(&models.Currency{}).Error; err != nil {
		cr.log.Error(constants.ErrExecQueryFromDB, err, slog.String("currencyCode", currencyCode))
		return err
	}

	cacheKey := fmt.Sprintf("currency:%s", currencyCode)
	if err := cr.cache.Del(context.Background(), cacheKey).Err(); err != nil {
		cr.log.Error(constants.ErrDeleteDataFromCache, err, slog.String("currencyCode", currencyCode))
		return err
	}

	return nil
}
