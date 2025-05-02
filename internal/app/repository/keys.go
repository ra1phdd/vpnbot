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

type Keys struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *cache.Cache
}

func NewKeys(log *logger.Logger, db *gorm.DB, cache *cache.Cache) *Keys {
	return &Keys{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (kr *Keys) GetAll(userID int64) (keys []*models.Key, err error) {
	cacheKey := fmt.Sprintf("key:user_id:%d", userID)
	if err = kr.cache.Get(cacheKey, keys); err == nil {
		kr.log.Debug("Returning keys from cache", slog.String("cache_key", cacheKey), slog.Int("count", len(keys)), slog.Int64("user_id", userID))
		return keys, nil
	}

	keys = make([]*models.Key, 0)
	if err = kr.db.Preload("Country").Where("user_id = ?", userID).Order("id DESC").Limit(1).First(&keys).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			kr.log.Debug("No keys found in database", slog.Int64("user_id", userID))
			return nil, nil
		}

		kr.log.Error("Failed to get keys from db", err, slog.Int64("user_id", userID))
		return nil, err
	}

	kr.cache.Set(cacheKey, keys, 15*time.Minute)
	kr.log.Debug("Returning keys from db", slog.Int("count", len(keys)), slog.Int64("user_id", userID))
	return keys, nil
}

func (kr *Keys) Get(countryID int, userID int64) (key *models.Key, err error) {
	cacheKey := fmt.Sprintf("key:user_id:%d:country_id:%d", userID, countryID)
	if err = kr.cache.Get(cacheKey, key); err == nil {
		kr.log.Debug("Returning key from cache", slog.String("cache_key", cacheKey), slog.Int("country_id", countryID), slog.Int64("user_id", userID))
		return key, nil
	}

	key = &models.Key{}
	if err = kr.db.Preload("Country").Where("country_id = ? AND user_id = ?", countryID, userID).Order("id DESC").First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			kr.log.Debug("Key not found in database", slog.Int("country_id", countryID), slog.Int64("user_id", userID))
			return nil, nil
		}

		kr.log.Error("Failed to get key from db", err, slog.Int("country_id", countryID), slog.Int64("user_id", userID))
		return nil, err
	}

	kr.cache.Set(cacheKey, key, 15*time.Minute)
	kr.log.Debug("Returning key from db", slog.Int("country_id", countryID), slog.Int64("user_id", userID))
	return key, nil
}

func (kr *Keys) Add(key *models.Key) error {
	if err := kr.db.Create(&key).Error; err != nil {
		kr.log.Error("Failed to create key in db", err, slog.Int("country_id", key.CountryID), slog.Int64("user_id", key.UserID))
		return err
	}

	kr.cache.Delete(fmt.Sprintf("key:user_id:%d", key.UserID), fmt.Sprintf("key:user_id:%d:country_id:%d", key.UserID, key.CountryID))
	kr.log.Debug("Added new key in db", slog.Int("country_id", key.CountryID), slog.Int64("user_id", key.UserID))
	return nil
}

func (kr *Keys) Update(countryID int, userID int64, newKey *models.Key) error {
	key, err := kr.Get(countryID, userID)
	if err != nil {
		kr.log.Error("Failed to get key for update", err, slog.Int("country_id", countryID), slog.Int64("user_id", userID))
		return err
	}

	tx := kr.db.Begin()
	if tx.Error != nil {
		kr.log.Error("Failed to begin transaction", tx.Error, slog.Int("country_id", countryID), slog.Int64("user_id", userID))
		return tx.Error
	}

	if err = updateField(kr.log, tx, key, "uuid", key.UUID, newKey.UUID); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(kr.log, tx, key, "speed_limit", key.SpeedLimit, newKey.SpeedLimit); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(kr.log, tx, key, "traffic_limit", key.TrafficLimit, newKey.TrafficLimit); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(kr.log, tx, key, "traffic_used", key.TrafficUsed, newKey.TrafficUsed); err != nil {
		tx.Rollback()
		return err
	}

	if err = tx.Commit().Error; err != nil {
		kr.log.Error("Failed to commit transaction", err, slog.Int("country_id", countryID), slog.Int64("user_id", userID))
		return err
	}

	kr.cache.Delete(fmt.Sprintf("key:user_id:%d", key.UserID), fmt.Sprintf("key:user_id:%d:country_id:%d", key.UserID, key.CountryID))
	kr.log.Debug("Successfully updated key", slog.Int("country_id", countryID), slog.Int64("user_id", userID), slog.Any("updatedFields", newKey))
	return nil
}

func (kr *Keys) UpdateIsActive(countryID int, userID int64, isActive bool) error {
	if err := kr.db.Model(&models.Key{}).Where("country_id = ? AND user_id = ?", countryID, userID).Update("is_active", isActive).Error; err != nil {
		kr.log.Error("Failed to update is_active", err, slog.Int("country_id", countryID), slog.Int64("user_id", userID), slog.Bool("is_active", isActive))
		return err
	}

	kr.cache.Delete(fmt.Sprintf("key:user_id:%d", userID), fmt.Sprintf("key:user_id:%d:country_id:%d", userID, countryID))
	kr.log.Debug("Successfully updated key", slog.Int("country_id", countryID), slog.Int64("user_id", userID), slog.Bool("is_active", isActive))
	return nil
}

func (kr *Keys) Delete(countryID int, userID int64) error {
	if err := kr.db.Where("country_id = ? AND user_id = ?", countryID, userID).Delete(&models.Key{}).Error; err != nil {
		kr.log.Error("Failed to delete key", err, slog.Int("country_id", countryID), slog.Int64("user_id", userID))
		return err
	}

	kr.cache.Delete(fmt.Sprintf("key:user_id:%d", userID), fmt.Sprintf("key:user_id:%d:country_id:%d", userID, countryID))
	kr.log.Debug("Deleted key from db", slog.Int("country_id", countryID), slog.Int64("user_id", userID))
	return nil
}
