package repository

import (
	"errors"
	"gorm.io/gorm"
	"log/slog"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/logger"
	"time"
)

type Currency struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *cache.Cache
}

func NewCurrency(log *logger.Logger, db *gorm.DB, cache *cache.Cache) *Currency {
	return &Currency{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (cr *Currency) GetAll() (currencies []*models.Currency, err error) {
	cacheKey := "currency:all"
	if err = cr.cache.Get(cacheKey, &currencies); err == nil {
		cr.log.Debug("Returning currencies from cache", slog.String("cache_key", cacheKey), slog.Int("count", len(currencies)))
		return currencies, nil
	}

	if err = cr.db.Find(&currencies).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cr.cache.Set(cacheKey, currencies, 15*time.Minute)
			cr.log.Debug("No currencies found in database")
			return nil, nil
		}

		cr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	cr.cache.Set(cacheKey, currencies, 15*time.Minute)
	cr.log.Debug("Returning currencies from db", slog.Int("count", len(currencies)))
	return currencies, nil
}

func (cr *Currency) Get(code string) (currency *models.Currency, err error) {
	cacheKey := "currency:" + code
	if err = cr.cache.Get(cacheKey, &currency); err == nil {
		cr.log.Debug("Returning currency from cache", slog.String("cache_key", cacheKey), slog.Any("currency", currency))
		return currency, nil
	}

	if err = cr.db.Where("code = ?", code).Order("id DESC").First(&currency).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cr.cache.Set(cacheKey, currency, 15*time.Minute)
			cr.log.Debug("Currency not found in database", slog.String("code", code))
			return nil, nil
		}

		cr.log.Error("Failed to get currency from db", err, slog.String("code", code))
		return nil, err
	}

	cr.cache.Set(cacheKey, currency, 15*time.Minute)
	cr.log.Debug("Returning currency from db", slog.String("code", code))
	return currency, nil
}

func (cr *Currency) GetIsBase() (currency *models.Currency, err error) {
	cacheKey := "currency:is_base"
	if err = cr.cache.Get(cacheKey, &currency); err == nil {
		cr.log.Debug("Returning currency from cache", slog.String("cache_key", cacheKey), slog.Any("currency", currency))
		return currency, nil
	}

	if err = cr.db.Where("is_base = ?", true).Order("id DESC").First(&currency).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cr.cache.Set(cacheKey, currency, 15*time.Minute)
			cr.log.Debug("Currency not found in database")
			return nil, nil
		}

		cr.log.Error("Failed to get currency from db", err)
		return nil, err
	}

	cr.cache.Set(cacheKey, currency, 15*time.Minute)
	cr.log.Debug("Returning currency from db")
	return currency, nil
}

func (cr *Currency) Add(currency *models.Currency) (uint, error) {
	if err := cr.db.Create(&currency).Error; err != nil {
		cr.log.Error("Failed to create currency in db", err, slog.Any("currency", currency))
		return 0, err
	}

	if currency.IsBase {
		cr.cache.Delete("currency:is_base")
	}
	cr.cache.Delete("currency:all")
	cr.log.Debug("Added new currency in db", slog.Any("currency", currency))
	return currency.ID, nil
}

func (cr *Currency) Update(code string, newCurrency *models.Currency) error {
	currency, err := cr.Get(code)
	if err != nil {
		cr.log.Error("Failed to get currency for update", err, slog.String("code", code))
		return err
	}

	tx := cr.db.Begin()
	if tx.Error != nil {
		cr.log.Error("Failed to begin transaction", tx.Error, slog.String("code", code))
		return tx.Error
	}

	if err = updateField(cr.log, tx, currency, "symbol", currency.Symbol, newCurrency.Symbol); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(cr.log, tx, currency, "name", currency.Name, newCurrency.Name); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(cr.log, tx, currency, "exchange_rate", currency.ExchangeRate, newCurrency.ExchangeRate); err != nil {
		tx.Rollback()
		return err
	}

	if err = tx.Commit().Error; err != nil {
		cr.log.Error("Failed to commit transaction", err, slog.String("code", code))
		return err
	}

	if currency.IsBase {
		cr.cache.Delete("currency:is_base")
	}
	cr.cache.Delete("currency:all", "currency:"+code)
	cr.log.Debug("Successfully updated currency", slog.String("code", code), slog.Any("updatedFields", newCurrency))
	return nil
}

func (cr *Currency) Delete(code string) error {
	currency, err := cr.Get(code)
	if err != nil {
		return err
	}

	if err := cr.db.Where("code = ?", code).Delete(&models.Currency{}).Error; err != nil {
		cr.log.Error("Failed to delete currency", err, slog.String("code", code))
		return err
	}

	if currency.IsBase {
		cr.cache.Delete("currency:is_base")
	}
	cr.cache.Delete("currency:all", "currency:"+code)
	cr.log.Debug("Deleted currency from db", slog.String("code", code))
	return nil
}
