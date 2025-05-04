package repository

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log/slog"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/logger"
	"time"
)

type Promocodes struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *cache.Cache

	Activations *PromocodesActivations
}

func NewPromocodes(log *logger.Logger, db *gorm.DB, cache *cache.Cache) *Promocodes {
	return &Promocodes{
		log:   log,
		db:    db,
		cache: cache,
		Activations: &PromocodesActivations{
			log:   log,
			db:    db,
			cache: cache,
		},
	}
}

func (pr *Promocodes) GetAll() (promocodes []*models.Promocode, err error) {
	cacheKey := "promocodes:all"
	if err = pr.cache.Get(cacheKey, &promocodes); err == nil {
		pr.log.Debug("Returning promocodes from cache", slog.String("cache_key", cacheKey), slog.Int("count", len(promocodes)))
		return promocodes, nil
	}

	if err = pr.db.Find(&promocodes).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pr.cache.Set(cacheKey, promocodes, 15*time.Minute)
			pr.log.Debug("No promocodes found in database")
			return nil, nil
		}

		pr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	pr.cache.Set(cacheKey, promocodes, 15*time.Minute)
	pr.log.Debug("Returning promocodes from db", slog.String("cache_key", cacheKey), slog.Int("count", len(promocodes)))
	return promocodes, nil
}

func (pr *Promocodes) GetAllActive() (promocodes []*models.Promocode, err error) {
	cacheKey := "promocodes:only_active"
	if err = pr.cache.Get(cacheKey, &promocodes); err == nil {
		pr.log.Debug("Returning promocodes from cache", slog.String("cache_key", cacheKey), slog.Int("count", len(promocodes)))
		return promocodes, nil
	}

	if err = pr.db.Where("is_active = ?", true).Find(&promocodes).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pr.cache.Set(cacheKey, promocodes, 15*time.Minute)
			pr.log.Debug("No promocodes found in database")
			return nil, nil
		}

		pr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	pr.cache.Set(cacheKey, promocodes, 15*time.Minute)
	pr.log.Debug("Returning promocodes from db", slog.String("cache_key", cacheKey), slog.Int("count", len(promocodes)))
	return promocodes, nil
}

func (pr *Promocodes) GetByID(id uint) (promocode *models.Promocode, err error) {
	cacheKey := fmt.Sprintf("promocodes:%d", id)
	if err = pr.cache.Get(cacheKey, &promocode); err == nil {
		pr.log.Debug("Returning promocode from cache", slog.String("cache_key", cacheKey), slog.Uint64("id", uint64(id)))
		return promocode, nil
	}

	if err = pr.db.Where("id = ?", id).First(&promocode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pr.cache.Set(cacheKey, promocode, 15*time.Minute)
			pr.log.Debug("Promocode not found in database", slog.Uint64("id", uint64(id)))
			return nil, nil
		}

		pr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	pr.cache.Set(cacheKey, promocode, 15*time.Minute)
	pr.log.Debug("Returning promocode from db", slog.Uint64("id", uint64(id)))
	return promocode, nil
}

func (pr *Promocodes) Get(code string) (promocode *models.Promocode, err error) {
	cacheKey := "promocodes:" + code
	if err = pr.cache.Get(cacheKey, &promocode); err == nil {
		pr.log.Debug("Returning promocode from cache", slog.String("cache_key", cacheKey), slog.String("code", code))
		return promocode, nil
	}

	if err = pr.db.Where("code = ?", code).First(&promocode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pr.cache.Set(cacheKey, promocode, 15*time.Minute)
			pr.log.Debug("Promocode not found in database", slog.String("code", code))
			return nil, nil
		}

		pr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	pr.cache.Set(cacheKey, promocode, 15*time.Minute)
	pr.log.Debug("Returning promocode from db", slog.String("code", code))
	return promocode, nil
}

func (pr *Promocodes) Add(promocode *models.Promocode) error {
	if err := pr.db.Create(&promocode).Error; err != nil {
		pr.log.Error("Failed to execute query from db", err, slog.String("code", promocode.Code))
		return err
	}

	pr.cache.Delete("promocodes:all", "promocodes:only_active")
	pr.log.Debug("Added new promocode in db", slog.String("code", promocode.Code))
	return nil
}

func (pr *Promocodes) UpdateByID(id uint, newPromocode *models.Promocode) error {
	promocode, err := pr.GetByID(id)
	if err != nil {
		pr.log.Error("Error executing the GetByCode function", err, slog.Uint64("id", uint64(id)))
		return err
	}

	tx := pr.db.Begin()
	if tx.Error != nil {
		pr.log.Error("Failed to begin transaction", tx.Error, slog.Uint64("id", uint64(id)))
		return tx.Error
	}

	if err = updateField(pr.log, tx, promocode, "code", promocode.Code, newPromocode.Code); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(pr.log, tx, promocode, "discount", promocode.Discount, newPromocode.Discount); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(pr.log, tx, promocode, "total_activations", promocode.TotalActivations, newPromocode.TotalActivations); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(pr.log, tx, promocode, "current_activations", promocode.CurrentActivations, newPromocode.CurrentActivations); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		pr.log.Error("Failed to commit transaction", err, slog.Uint64("id", uint64(id)))
		return err
	}

	pr.cache.Delete("promocodes:all", "promocodes:only_active", fmt.Sprintf("promocodes:%d", id), "promocodes:"+promocode.Code)
	pr.log.Debug("Successfully updated newPromocode", slog.Uint64("id", uint64(id)))
	return nil
}

func (pr *Promocodes) Update(code string, newPromocode *models.Promocode) error {
	promocode, err := pr.Get(code)
	if err != nil {
		pr.log.Error("Error executing the GetByCode function", err, slog.String("code", code))
		return err
	}

	tx := pr.db.Begin()
	if tx.Error != nil {
		pr.log.Error("Failed to begin transaction", tx.Error, slog.String("code", code))
		return tx.Error
	}

	if err = updateField(pr.log, tx, promocode, "discount", promocode.Discount, newPromocode.Discount); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(pr.log, tx, promocode, "total_activations", promocode.TotalActivations, newPromocode.TotalActivations); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(pr.log, tx, promocode, "current_activations", promocode.CurrentActivations, newPromocode.CurrentActivations); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		pr.log.Error("Failed to commit transaction", err, slog.String("code", code))
		return err
	}

	pr.cache.Delete("promocodes:all", "promocodes:only_active", fmt.Sprintf("promocodes:%d", promocode.ID), "promocodes:"+promocode.Code)
	pr.log.Debug("Successfully updated newPromocode", slog.String("code", code))
	return nil
}

func (pr *Promocodes) UpdateOnlyNewUsers(code string, onlyNewUsers bool) error {
	promocode, err := pr.Get(code)
	if err != nil {
		return err
	}

	if err := pr.db.Model(&models.Promocode{}).Where("code = ?", code).Update("only_new_users", onlyNewUsers).Error; err != nil {
		pr.log.Error("Failed to execute query from db", err, slog.String("code", code))
		return err
	}

	pr.cache.Delete("promocodes:all", "promocodes:only_active", fmt.Sprintf("promocodes:%d", promocode.ID), "promocodes:"+code)
	pr.log.Debug("Successfully updated only_new_users", slog.String("code", code), slog.Bool("only_new", onlyNewUsers))
	return nil
}

func (pr *Promocodes) UpdateIsActive(code string, isActive bool) error {
	promocode, err := pr.Get(code)
	if err != nil {
		return err
	}

	if err := pr.db.Model(&models.Promocode{}).Where("code = ?", code).Update("is_active", isActive).Error; err != nil {
		pr.log.Error("Failed to execute query from db", err, slog.String("code", code))
		return err
	}

	pr.cache.Delete("promocodes:all", "promocodes:only_active", fmt.Sprintf("promocodes:%d", promocode.ID), "promocodes:"+code)
	pr.log.Debug("Successfully updated is_active", slog.String("code", code), slog.Bool("is_active", isActive))
	return nil
}

func (pr *Promocodes) IncrementActivationsByID(id uint) error {
	promocode, err := pr.GetByID(id)
	if err != nil {
		return err
	}

	result := pr.db.Model(&models.Promocode{}).
		Where("id = ?", id).
		Update("current_activations", gorm.Expr("current_activations + ?", 1))
	if result.Error != nil {
		pr.log.Error("Failed to increment activations", result.Error, slog.Uint64("id", uint64(id)))
		return result.Error
	}

	if result.RowsAffected == 0 {
		return constants.ErrUserNotFound
	}

	pr.cache.Delete("promocodes:all", "promocodes:only_active", fmt.Sprintf("promocodes:%d", id), "promocodes:"+promocode.Code)
	pr.log.Debug("Incremented promocode activations", slog.Uint64("id", uint64(id)))
	return nil
}

func (pr *Promocodes) Delete(code string) error {
	promocode, err := pr.Get(code)
	if err != nil {
		return err
	}

	if err := pr.db.Where("code = ?", code).Delete(&models.Promocode{}).Error; err != nil {
		pr.log.Error("Failed to execute query from db", err, slog.String("code", code))
		return err
	}

	pr.cache.Delete("promocodes:all", "promocodes:only_active", fmt.Sprintf("promocodes:%d", promocode.ID), "promocodes:"+code)
	pr.log.Debug("Deleted promocode from db", slog.String("code", code))
	return nil
}
