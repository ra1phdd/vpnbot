package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/logger"
	"time"
)

type Promocodes struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *redis.Client
}

func NewPromocodes(log *logger.Logger, db *gorm.DB, cache *redis.Client) *Promocodes {
	return &Promocodes{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (pr *Promocodes) GetAll(includeInactive bool) ([]models.Promocode, error) {
	cacheKey := "promocodes:only_active"
	if includeInactive {
		cacheKey = "promocodes:all"
	}

	cacheValue, err := pr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		pr.log.Error(constants.ErrGetDataFromCache, err)
		return nil, err
	}

	var promocodes []models.Promocode
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &promocodes); err != nil {
			pr.log.Error(constants.ErrUnmarshalDataFromJSON, err)
			return nil, err
		}
		return promocodes, nil
	}

	query := pr.db
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Find(&promocodes).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		pr.log.Error(constants.ErrGetDataFromDB, err)
		return nil, err
	}

	jsonData, err := json.Marshal(promocodes)
	if err != nil {
		pr.log.Error(constants.ErrMarshalDataToJSON, err)
		return nil, err
	}
	if err := pr.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		pr.log.Error(constants.ErrSetDataToCache, err)
		return nil, err
	}

	return promocodes, nil
}

func (pr *Promocodes) GetByCode(code string) (models.Promocode, error) {
	cacheKey := fmt.Sprintf("promocodes:%s", code)
	cacheValue, err := pr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		pr.log.Error(constants.ErrGetDataFromCache, err)
		return models.Promocode{}, err
	}

	var promocode models.Promocode
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &promocode); err != nil {
			pr.log.Error(constants.ErrUnmarshalDataFromJSON, err)
			return models.Promocode{}, err
		}
		return promocode, nil
	}

	if err := pr.db.Where("code = ?", code).First(&promocode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Promocode{}, nil
		}

		pr.log.Error(constants.ErrGetDataFromDB, err)
		return promocode, err
	}

	jsonData, err := json.Marshal(promocode)
	if err != nil {
		pr.log.Error(constants.ErrMarshalDataToJSON, err)
		return models.Promocode{}, err
	}
	if err := pr.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		pr.log.Error(constants.ErrSetDataToCache, err)
		return models.Promocode{}, err
	}

	return promocode, nil
}

func (pr *Promocodes) Add(promocode models.Promocode) error {
	if err := pr.db.Create(&promocode).Error; err != nil {
		pr.log.Error(constants.ErrExecQueryFromDB, err)
		return err
	}
	return nil
}

func (pr *Promocodes) Update(code string, promocode models.Promocode) error {
	promocodeOld, err := pr.GetByCode(code)
	if err != nil {
		pr.log.Error("Error executing the GetByCode function", err)
		return err
	}

	tx := pr.db.Begin()
	if tx.Error != nil {
		pr.log.Error(constants.ErrBeginTx, tx.Error)
		return tx.Error
	}

	if promocode.Discount != promocodeOld.Discount && promocode.Discount > 0 {
		if err := tx.Model(&promocodeOld).Where("code = ?", code).Update("discount", promocode.Discount).Error; err != nil {
			tx.Rollback()
			pr.log.Error(constants.ErrExecQueryFromDB, err)
			return err
		}
	}

	if promocode.TotalActivations != nil && promocodeOld.TotalActivations != nil && *promocode.TotalActivations != *promocodeOld.TotalActivations {
		if err := tx.Model(&promocodeOld).Where("code = ?", code).Update("total_activations", promocode.TotalActivations).Error; err != nil {
			tx.Rollback()
			pr.log.Error(constants.ErrExecQueryFromDB, err)
			return err
		}
	}

	if promocode.CurrentActivations > 0 && promocode.CurrentActivations != promocodeOld.CurrentActivations {
		if err := tx.Model(&promocodeOld).Where("code = ?", code).Update("current_activations", promocode.CurrentActivations).Error; err != nil {
			tx.Rollback()
			pr.log.Error(constants.ErrExecQueryFromDB, err)
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		pr.log.Error(constants.ErrCommitTx, err)
		return err
	}

	cacheKeys := []string{
		fmt.Sprintf("promocodes:%s", code),
		"promocodes:all",
		"promocodes:only_active",
	}
	for _, key := range cacheKeys {
		if err := pr.cache.Del(context.Background(), key).Err(); err != nil {
			pr.log.Error(constants.ErrDeleteDataFromCache, err)
			return err
		}
	}

	return nil
}

func (pr *Promocodes) UpdateOnlyNewUsers(code string, onlyNew bool) error {
	if err := pr.db.Model(&models.Promocode{}).Where("code = ?", code).Update("only_new_users", onlyNew).Error; err != nil {
		pr.log.Error(constants.ErrExecQueryFromDB, err)
		return err
	}

	cacheKeys := []string{
		fmt.Sprintf("promocodes:%s", code),
		"promocodes:all",
		"promocodes:only_active",
	}
	for _, key := range cacheKeys {
		if err := pr.cache.Del(context.Background(), key).Err(); err != nil {
			pr.log.Error(constants.ErrDeleteDataFromCache, err)
			return err
		}
	}
	return nil
}

func (pr *Promocodes) UpdateIsActive(code string, isActive bool) error {
	if err := pr.db.Model(&models.Promocode{}).Where("code = ?", code).Update("is_active", isActive).Error; err != nil {
		pr.log.Error(constants.ErrExecQueryFromDB, err)
		return err
	}

	cacheKeys := []string{
		fmt.Sprintf("promocodes:%s", code),
		"promocodes:all",
		"promocodes:only_active",
	}
	for _, key := range cacheKeys {
		if err := pr.cache.Del(context.Background(), key).Err(); err != nil {
			pr.log.Error(constants.ErrDeleteDataFromCache, err)
			return err
		}
	}
	return nil
}

func (pr *Promocodes) Delete(code string) error {
	if err := pr.db.Where("code = ?", code).Delete(&models.Promocode{}).Error; err != nil {
		pr.log.Error(constants.ErrExecQueryFromDB, err)
		return err
	}

	cacheKeys := []string{
		fmt.Sprintf("promocodes:%s", code),
		"promocodes:all",
		"promocodes:only_active",
	}
	for _, key := range cacheKeys {
		if err := pr.cache.Del(context.Background(), key).Err(); err != nil {
			pr.log.Error(constants.ErrDeleteDataFromCache, err)
			return err
		}
	}

	return nil
}
