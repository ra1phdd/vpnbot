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

type Country struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *cache.Cache
}

func NewCountry(log *logger.Logger, db *gorm.DB, cache *cache.Cache) *Country {
	return &Country{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (cr *Country) GetAll() (countries []*models.Country, err error) {
	cacheKey := "country:all"
	if err = cr.cache.Get(cacheKey, countries); err == nil {
		cr.log.Debug("Returning countries from cache", slog.String("cache_key", cacheKey), slog.Int("count", len(countries)))
		return countries, nil
	}

	countries = make([]*models.Country, 0)
	if err = cr.db.Find(&countries).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cr.log.Debug("No countries found in database")
			return nil, nil
		}

		cr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	cr.cache.Set(cacheKey, countries, 15*time.Minute)
	cr.log.Debug("Returning countries from db", slog.Int("count", len(countries)))
	return countries, nil
}

func (cr *Country) Get(code string) (country *models.Country, err error) {
	cacheKey := "country:" + code
	if err = cr.cache.Get(cacheKey, country); err == nil {
		cr.log.Debug("Returning country from cache", slog.String("cache_key", cacheKey), slog.Any("country", country))
		return country, nil
	}

	country = &models.Country{}
	if err = cr.db.Where("code = ?", code).Order("id DESC").First(&country).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cr.log.Debug("Country not found in database", slog.String("code", code))
			return nil, nil
		}

		cr.log.Error("Failed to get country from db", err, slog.String("code", code))
		return nil, err
	}

	cr.cache.Set(cacheKey, country, 15*time.Minute)
	cr.log.Debug("Returning countries from db", slog.String("code", code))
	return country, nil
}

func (cr *Country) Add(country *models.Country) (int, error) {
	if err := cr.db.Create(&country).Error; err != nil {
		cr.log.Error("Failed to create country in db", err, slog.String("code", country.Code))
		return 0, err
	}

	cr.cache.Delete("country:all")
	cr.log.Debug("Added new country in db", slog.String("code", country.Code), slog.Int("id", country.ID))
	return country.ID, nil
}

func (cr *Country) Update(code string, newCountry *models.Country) error {
	country, err := cr.Get(code)
	if err != nil {
		cr.log.Error("Failed to get country for update", err, slog.String("code", code))
		return err
	}

	tx := cr.db.Begin()
	if tx.Error != nil {
		cr.log.Error("Failed to begin transaction", tx.Error, slog.String("code", code))
		return tx.Error
	}

	if err = updateField(cr.log, tx, country, "emoji", country.Emoji, newCountry.Emoji); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(cr.log, tx, country, "name_ru", country.NameRU, newCountry.NameRU); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(cr.log, tx, country, "name_en", country.NameEN, newCountry.NameEN); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(cr.log, tx, country, "domain", country.Domain, newCountry.Domain); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(cr.log, tx, country, "private_key", country.PrivateKey, newCountry.PrivateKey); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(cr.log, tx, country, "public_key", country.PublicKey, newCountry.PublicKey); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(cr.log, tx, country, "dest", country.Dest, newCountry.Dest); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(cr.log, tx, country, "server_names", country.ServerNames, newCountry.ServerNames); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(cr.log, tx, country, "short_ids", country.ShortIDs, newCountry.ShortIDs); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		cr.log.Error("Failed to commit transaction", err, slog.String("code", code))
		return err
	}

	cr.cache.Delete("country:all", "country:"+code)
	cr.log.Debug("Successfully updated country", slog.String("code", code))
	return nil
}

func (cr *Country) Delete(code string) error {
	if err := cr.db.Where("code = ?", code).Delete(&models.Country{}).Error; err != nil {
		cr.log.Error("Failed to delete country", err, slog.String("code", code))
		return err
	}

	cr.cache.Delete("country:all", "country:"+code)
	cr.log.Debug("Deleted country from db", slog.String("code", code))
	return nil
}
