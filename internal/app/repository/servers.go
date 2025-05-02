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

type Servers struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *cache.Cache
}

func NewServers(log *logger.Logger, db *gorm.DB, cache *cache.Cache) *Servers {
	return &Servers{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (sr *Servers) GetAll() (servers []*models.Server, err error) {
	cacheKey := "servers:all"
	if err = sr.cache.Get(cacheKey, servers); err == nil {
		sr.log.Debug("Returning servers from cache", slog.String("cache_key", cacheKey), slog.Int("count", len(servers)))
		return servers, nil
	}

	if err = sr.db.Find(&servers).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sr.log.Debug("No servers found in database")
			return nil, nil
		}

		sr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	sr.cache.Set(cacheKey, servers, 15*time.Minute)
	sr.log.Debug("Returning servers from db", slog.String("cache_key", cacheKey), slog.Int("count", len(servers)))
	return servers, nil
}

func (sr *Servers) GetAllByCountryID(countryID int) (servers []*models.Server, err error) {
	cacheKey := fmt.Sprintf("servers:country_id:%d", countryID)
	if err = sr.cache.Get(cacheKey, servers); err == nil {
		sr.log.Debug("Returning servers from cache", slog.String("cache_key", cacheKey), slog.Int("count", len(servers)))
		return servers, nil
	}

	if err = sr.db.Preload("Country").Where("country_id = ?", countryID).Find(&servers).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sr.log.Debug("No servers found in database", slog.Int("countryID", countryID))
			return nil, nil
		}

		sr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	sr.cache.Set(cacheKey, servers, 15*time.Minute)
	sr.log.Debug("Returning servers from db", slog.String("cache_key", cacheKey), slog.Int("count", len(servers)))
	return servers, nil
}

func (sr *Servers) Get(id int) (server *models.Server, err error) {
	cacheKey := fmt.Sprintf("servers:id:%d", id)
	if err = sr.cache.Get(cacheKey, server); err == nil {
		sr.log.Debug("Returning server from cache", slog.String("cache_key", cacheKey), slog.Int("id", id))
		return server, nil
	}

	if err = sr.db.First(server, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sr.log.Debug("Server not found in database", slog.Int("id", id))
			return nil, nil
		}

		sr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	sr.cache.Set(cacheKey, server, 15*time.Minute)
	sr.log.Debug("Returning server from db", slog.Int("id", id))
	return server, nil
}

func (sr *Servers) Add(server *models.Server) error {
	if err := sr.db.Create(&server).Error; err != nil {
		sr.log.Error("Failed to execute query from db", err, slog.Int("id", server.ID))
		return err
	}

	sr.cache.Delete("servers:all")
	sr.log.Debug("Added new server in db", slog.Int("id", server.ID))
	return nil
}

func (sr *Servers) Update(id int, newServer *models.Server) error {
	server, err := sr.Get(id)
	if err != nil {
		sr.log.Error("Failed to execute query from db", err, slog.Int("id", id))
		return err
	}

	tx := sr.db.Begin()
	if tx.Error != nil {
		sr.log.Error("Failed to begin transaction", tx.Error)
		return tx.Error
	}

	if err = updateField(sr.log, tx, server, "ip", server.IP, newServer.IP); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(sr.log, tx, server, "port", server.Port, newServer.Port); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(sr.log, tx, server, "country_id", server.CountryID, newServer.CountryID); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(sr.log, tx, server, "channel_speed", server.ChannelSpeed, newServer.ChannelSpeed); err != nil {
		tx.Rollback()
		return err
	}

	if err = tx.Commit().Error; err != nil {
		sr.log.Error("Failed to commit transaction", err)
		return err
	}

	sr.cache.Delete("servers:all", fmt.Sprintf("servers:id:%d", id), fmt.Sprintf("servers:country_id:%d", server.CountryID))
	sr.log.Debug("Successfully updated newServer", slog.Int("id", id))
	return nil
}

func (sr *Servers) Delete(id int) error {
	server, err := sr.Get(id)
	if err != nil {
		sr.log.Error("Failed to execute query from db", err, slog.Int("id", id))
		return err
	}

	if err = sr.db.Delete(&models.Server{}, id).Error; err != nil {
		sr.log.Error("Failed to delete server from db", err)
		return err
	}

	sr.cache.Delete("servers:all", fmt.Sprintf("servers:id:%d", id), fmt.Sprintf("servers:country_id:%d", server.CountryID))
	sr.log.Debug("Deleted server from db", slog.Int("id", id))
	return nil
}
