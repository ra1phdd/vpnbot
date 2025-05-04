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

type PromocodesActivations struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *cache.Cache
}

func (pr *PromocodesActivations) GetAll() (activations []*models.PromocodeActivations, err error) {
	cacheKey := "promocode_activations:all"
	if err = pr.cache.Get(cacheKey, &activations); err == nil {
		pr.log.Debug("Returning promocode Activations from cache", slog.String("cache_key", cacheKey), slog.Int("count", len(activations)))
		return activations, nil
	}

	if err = pr.db.Preload("Promocode").Preload("User").Find(&activations).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pr.cache.Set(cacheKey, activations, 15*time.Minute)
			pr.log.Debug("No promocode Activations found in database")
			return nil, nil
		}

		pr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	pr.cache.Set(cacheKey, activations, 15*time.Minute)
	pr.log.Debug("Returning promocode Activations from db", slog.String("cache_key", cacheKey), slog.Int("count", len(activations)))
	return activations, nil
}

func (pr *PromocodesActivations) GetByUserID(userID int64) (activations []*models.PromocodeActivations, err error) {
	cacheKey := fmt.Sprintf("promocode_activations:user:%d", userID)
	if err = pr.cache.Get(cacheKey, &activations); err == nil {
		pr.log.Debug("Returning user promocode Activations from cache", slog.String("cache_key", cacheKey), slog.Int64("user_id", userID), slog.Int("count", len(activations)))
		return activations, nil
	}

	if err = pr.db.Preload("Promocode").Preload("User").Where("user_id = ?", userID).Find(&activations).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pr.cache.Set(cacheKey, activations, 15*time.Minute)
			pr.log.Debug("No promocode Activations found for user", slog.Int64("user_id", userID))
			return nil, nil
		}

		pr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	pr.cache.Set(cacheKey, activations, 15*time.Minute)
	pr.log.Debug("Returning user promocode Activations from db", slog.String("cache_key", cacheKey), slog.Int64("user_id", userID), slog.Int("count", len(activations)))
	return activations, nil
}

func (pr *PromocodesActivations) GetByPromocodeID(promocodeID uint) (activations []*models.PromocodeActivations, err error) {
	cacheKey := fmt.Sprintf("promocode_activations:promocode:%d", promocodeID)
	if err = pr.cache.Get(cacheKey, &activations); err == nil {
		pr.log.Debug("Returning promocode Activations from cache", slog.String("cache_key", cacheKey), slog.Uint64("promocode_id", uint64(promocodeID)), slog.Int("count", len(activations)))
		return activations, nil
	}

	if err = pr.db.Preload("Promocode").Preload("User").Where("promocode_id = ?", promocodeID).Find(&activations).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pr.cache.Set(cacheKey, activations, 15*time.Minute)
			pr.log.Debug("No Activations found for promocode", slog.Uint64("promocode_id", uint64(promocodeID)))
			return nil, nil
		}

		pr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	pr.cache.Set(cacheKey, activations, 15*time.Minute)
	pr.log.Debug("Returning promocode Activations from db", slog.String("cache_key", cacheKey), slog.Uint64("promocode_id", uint64(promocodeID)), slog.Int("count", len(activations)))
	return activations, nil
}

func (pr *PromocodesActivations) GetByID(id uint) (activation *models.PromocodeActivations, err error) {
	cacheKey := fmt.Sprintf("promocode_activations:%d", id)
	if err = pr.cache.Get(cacheKey, &activation); err == nil {
		pr.log.Debug("Returning user promocode Activations from cache", slog.String("cache_key", cacheKey), slog.Uint64("id", uint64(id)))
		return activation, nil
	}

	if err = pr.db.Preload("Promocode").Preload("User").Where("id = ?", id).Find(&activation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pr.cache.Set(cacheKey, activation, 15*time.Minute)
			pr.log.Debug("No promocode Activations found for user", slog.Uint64("id", uint64(id)))
			return nil, nil
		}

		pr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	pr.cache.Set(cacheKey, activation, 15*time.Minute)
	pr.log.Debug("Returning user promocode Activations from db", slog.String("cache_key", cacheKey), slog.Uint64("id", uint64(id)))
	return activation, nil
}

func (pr *PromocodesActivations) Get(promocodeID uint, userID int64) (activation *models.PromocodeActivations, err error) {
	cacheKey := fmt.Sprintf("promocode_activations:promocode:%d:user:%d", promocodeID, userID)
	if err = pr.cache.Get(cacheKey, &activation); err == nil {
		pr.log.Debug("Returning user promocode Activations from cache", slog.String("cache_key", cacheKey), slog.Int64("user_id", userID), slog.Uint64("promocode_id", uint64(promocodeID)))
		return activation, nil
	}

	if err = pr.db.Preload("Promocode").Preload("User").Where("promocode_id = ? AND user_id = ?", promocodeID, userID).Find(&activation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pr.cache.Set(cacheKey, activation, 15*time.Minute)
			pr.log.Debug("No promocode Activations found for user", slog.Int64("user_id", userID))
			return nil, nil
		}

		pr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	pr.cache.Set(cacheKey, activation, 15*time.Minute)
	pr.log.Debug("Returning user promocode Activations from db", slog.String("cache_key", cacheKey), slog.Int64("user_id", userID))
	return activation, nil
}

func (pr *PromocodesActivations) Add(activation *models.PromocodeActivations) error {
	if err := pr.db.Create(&activation).Error; err != nil {
		pr.log.Error("Failed to execute query from db", err, slog.Uint64("promocode_id", uint64(activation.PromocodeID)), slog.Int64("user_id", activation.UserID))
		return err
	}

	pr.cache.Delete("promocode_activations:all", fmt.Sprintf("promocode_activations:%d", activation.ID), fmt.Sprintf("promocode_activations:promocode:%d", activation.PromocodeID), fmt.Sprintf("promocode_activations:user:%d", activation.UserID), fmt.Sprintf("promocode_activations:promocode:%d:user:%d", activation.PromocodeID, activation.UserID))
	pr.log.Debug("Added new promocode activation in db", slog.Uint64("promocode_id", uint64(activation.PromocodeID)), slog.Int64("user_id", activation.UserID))
	return nil
}

func (pr *PromocodesActivations) Delete(id uint) error {
	activation, err := pr.GetByID(id)
	if err != nil {
		return err
	}

	if err := pr.db.Where("id = ?", id).Delete(&models.PromocodeActivations{}).Error; err != nil {
		pr.log.Error("Failed to execute query from db", err, slog.Uint64("id", uint64(id)))
		return err
	}

	pr.cache.Delete("promocode_activations:all", fmt.Sprintf("promocode_activations:%d", id), fmt.Sprintf("promocode_activations:promocode:%d", activation.PromocodeID), fmt.Sprintf("promocode_activations:user:%d", activation.UserID), fmt.Sprintf("promocode_activations:promocode:%d:user:%d", activation.PromocodeID, activation.UserID))
	pr.log.Debug("Deleted promocode activation from db", slog.Uint64("id", uint64(id)))
	return nil
}
