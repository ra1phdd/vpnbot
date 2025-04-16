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

type Country struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *redis.Client
}

func NewCountry(log *logger.Logger, db *gorm.DB, cache *redis.Client) *Country {
	return &Country{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (cr *Country) GetAll() ([]models.Country, error) {
	cacheKey := "country:all"
	cacheValue, err := cr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		cr.log.Error(constants.ErrGetDataFromCache, err)
		return nil, err
	}

	var countries []models.Country
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &countries); err != nil {
			cr.log.Error(constants.ErrUnmarshalDataFromJSON, err)
			return nil, err
		}
		return countries, nil
	}

	if err := cr.db.Find(&countries).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		cr.log.Error(constants.ErrGetDataFromDB, err)
		return nil, err
	}

	jsonData, err := json.Marshal(countries)
	if err != nil {
		cr.log.Error(constants.ErrMarshalDataToJSON, err)
		return nil, err
	}
	if err := cr.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		cr.log.Error(constants.ErrSetDataToCache, err)
		return nil, err
	}

	return countries, nil
}

func (cr *Country) Get(countryCode string) (models.Country, error) {
	cacheKey := fmt.Sprintf("country:%s", countryCode)
	cacheValue, err := cr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		cr.log.Error(constants.ErrGetDataFromCache, err, slog.String("countryCode", countryCode))
		return models.Country{}, err
	}

	var country models.Country
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &country); err != nil {
			cr.log.Error(constants.ErrUnmarshalDataFromJSON, err, slog.String("countryCode", countryCode))
			return models.Country{}, err
		}
		return country, nil
	}

	if err := cr.db.Where("country_code = ?", countryCode).First(&country).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Country{}, nil
		}

		cr.log.Error(constants.ErrGetDataFromDB, err, slog.String("countryCode", countryCode))
		return models.Country{}, err
	}

	jsonData, err := json.Marshal(country)
	if err != nil {
		cr.log.Error(constants.ErrMarshalDataToJSON, err, slog.String("countryCode", countryCode))
		return models.Country{}, err
	}
	if err := cr.cache.Set(context.Background(), cacheKey, jsonData, 0).Err(); err != nil {
		cr.log.Error(constants.ErrSetDataToCache, err, slog.String("countryCode", countryCode))
		return models.Country{}, err
	}

	return country, nil
}

func (cr *Country) Add(country models.Country) (int, error) {
	if err := cr.db.Create(&country).Error; err != nil {
		cr.log.Error(constants.ErrExecQueryFromDB, err, slog.Any("country", country))
		return 0, err
	}

	return country.ID, nil
}

func (cr *Country) Delete(countryCode string) error {
	if err := cr.db.Where("country_code = ?", countryCode).Delete(&models.Country{}).Error; err != nil {
		cr.log.Error(constants.ErrExecQueryFromDB, err)
		return err
	}

	cacheKey := fmt.Sprintf("country:%s", countryCode)
	if err := cr.cache.Del(context.Background(), cacheKey).Err(); err != nil {
		cr.log.Error(constants.ErrDeleteDataFromCache, err)
		return err
	}

	return nil
}
