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

type Servers struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *redis.Client
}

func NewServers(log *logger.Logger, db *gorm.DB, cache *redis.Client) *Servers {
	return &Servers{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (sr *Servers) GetAll() ([]models.Server, error) {
	cacheKey := "servers:all"
	cacheValue, err := sr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		sr.log.Error("error getting data from cache", err)
		return nil, err
	}

	var servers []models.Server
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &servers); err != nil {
			sr.log.Error("error unmarshaling JSON", err)
			return nil, err
		}
		return servers, nil
	}

	if err := sr.db.Find(&servers).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		sr.log.Error("error fetching servers from DB", err)
		return nil, err
	}

	jsonData, err := json.Marshal(servers)
	if err != nil {
		sr.log.Error("error marshaling servers", err)
		return nil, err
	}
	if err := sr.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		sr.log.Error("error setting cache", err)
		return nil, err
	}

	return servers, nil
}

func (sr *Servers) GetById(id int) (models.Server, error) {
	cacheKey := fmt.Sprintf("servers:id:%d", id)
	cacheValue, err := sr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		sr.log.Error("error getting data from cache", err)
		return models.Server{}, err
	}

	var server models.Server
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &server); err != nil {
			sr.log.Error("error unmarshaling server", err)
			return models.Server{}, err
		}
		return server, nil
	}

	if err := sr.db.First(&server, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Server{}, nil
		}

		sr.log.Error("error fetching server from DB", err)
		return models.Server{}, err
	}

	jsonData, err := json.Marshal(server)
	if err != nil {
		sr.log.Error("error marshaling server", err)
		return models.Server{}, err
	}
	if err := sr.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		sr.log.Error("error setting cache", err)
		return models.Server{}, err
	}

	return server, nil
}

func (sr *Servers) GetByIP(ip string) (models.Server, error) {
	cacheKey := fmt.Sprintf("servers:ip:%s", ip)
	cacheValue, err := sr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		sr.log.Error("error getting data from cache", err)
		return models.Server{}, err
	}

	var server models.Server
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &server); err != nil {
			sr.log.Error("error unmarshaling server", err)
			return models.Server{}, err
		}
		return server, nil
	}

	if err := sr.db.Where("ip = ?", ip).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Server{}, nil
		}

		sr.log.Error("error fetching server from DB", err)
		return models.Server{}, err
	}

	jsonData, err := json.Marshal(server)
	if err != nil {
		sr.log.Error("error marshaling server", err)
		return models.Server{}, err
	}
	if err := sr.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		sr.log.Error("error setting cache", err)
		return models.Server{}, err
	}

	return server, nil
}

func (sr *Servers) GetByСС(countryCode string) ([]models.Server, error) {
	cacheKey := fmt.Sprintf("servers:country_code:%s", countryCode)
	cacheValue, err := sr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		sr.log.Error("error getting data from cache", err)
		return nil, err
	}

	var servers []models.Server
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &servers); err != nil {
			sr.log.Error("error unmarshaling servers", err)
			return nil, err
		}
		return servers, nil
	}

	if err := sr.db.Joins("JOIN countries ON countries.id = servers.country_id").Where("countries.country_code = ?", countryCode).Find(&servers).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		sr.log.Error("error fetching servers from DB", err)
		return nil, err
	}

	jsonData, err := json.Marshal(servers)
	if err != nil {
		sr.log.Error("error marshaling servers", err)
		return nil, err
	}
	if err := sr.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		sr.log.Error("error setting cache", err)
		return nil, err
	}

	return servers, nil
}

func (sr *Servers) Add(server models.Server) error {
	if err := sr.db.Create(&server).Error; err != nil {
		sr.log.Error("error adding server", err)
		return err
	}
	return nil
}

func (sr *Servers) Update(id int, server models.Server) error {
	serverOld, err := sr.GetById(id)
	if err != nil {
		return err
	}

	tx := sr.db.Begin()
	if tx.Error != nil {
		sr.log.Error("error starting transaction", tx.Error)
		return tx.Error
	}

	if server.IP != "" && server.IP != serverOld.IP {
		if err := tx.Model(&serverOld).Where("id = ?", id).Update("ip", server.IP).Error; err != nil {
			tx.Rollback()
			sr.log.Error("error updating IP", err)
			return err
		}
	}

	if server.Port != 0 && server.Port != serverOld.Port {
		if err := tx.Model(&serverOld).Where("id = ?", id).Update("port", server.Port).Error; err != nil {
			tx.Rollback()
			sr.log.Error("error updating port", err)
			return err
		}
	}

	if server.CountryID != 0 && server.CountryID != serverOld.CountryID {
		if err := tx.Model(&serverOld).Where("id = ?", id).Update("country_id", server.CountryID).Error; err != nil {
			tx.Rollback()
			sr.log.Error("error updating country_id", err)
			return err
		}
	}

	if server.ChannelSpeed != 0 && server.ChannelSpeed != serverOld.ChannelSpeed {
		if err := tx.Model(&serverOld).Where("id = ?", id).Update("channel_speed", server.ChannelSpeed).Error; err != nil {
			tx.Rollback()
			sr.log.Error("error updating channel_speed", err)
			return err
		}
	}

	if server.PrivateKey != "" && server.PrivateKey != serverOld.PrivateKey {
		if err := tx.Model(&serverOld).Where("id = ?", id).Update("private_key", server.PrivateKey).Error; err != nil {
			tx.Rollback()
			sr.log.Error("error updating private_key", err)
			return err
		}
	}

	if server.PublicKey != "" && server.PublicKey != serverOld.PublicKey {
		if err := tx.Model(&serverOld).Where("id = ?", id).Update("public_key", server.PublicKey).Error; err != nil {
			tx.Rollback()
			sr.log.Error("error updating public_key", err)
			return err
		}
	}

	if server.Dest != "" && server.Dest != serverOld.Dest {
		if err := tx.Model(&serverOld).Where("id = ?", id).Update("dest", server.Dest).Error; err != nil {
			tx.Rollback()
			sr.log.Error("error updating dest", err)
			return err
		}
	}

	if server.ServerNames != "" && server.ServerNames != serverOld.ServerNames {
		if err := tx.Model(&serverOld).Where("id = ?", id).Update("server_names", server.ServerNames).Error; err != nil {
			tx.Rollback()
			sr.log.Error("error updating server_names", err)
			return err
		}
	}

	if server.ShortIDs != "" && server.ShortIDs != serverOld.ShortIDs {
		if err := tx.Model(&serverOld).Where("id = ?", id).Update("short_ids", server.ShortIDs).Error; err != nil {
			tx.Rollback()
			sr.log.Error("error updating short_ids", err)
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		sr.log.Error("error committing transaction", err)
		return err
	}

	cacheKeys := []string{
		fmt.Sprintf("servers:id:%d", serverOld.ID),
		fmt.Sprintf("servers:ip:%s", serverOld.IP),
		fmt.Sprintf("servers:country_id:%d", serverOld.CountryID),
	}
	for _, key := range cacheKeys {
		if err := sr.cache.Del(context.Background(), key).Err(); err != nil {
			sr.log.Error(constants.ErrDeleteDataFromCache, err)
			return err
		}
	}

	return nil
}

func (sr *Servers) Delete(id int) error {
	server, err := sr.GetById(id)
	if err != nil {
		return err
	}

	if err := sr.db.Delete(&models.Server{}, id).Error; err != nil {
		sr.log.Error("error deleting server", err)
		return err
	}

	cacheKeys := []string{
		fmt.Sprintf("servers:id:%d", server.ID),
		fmt.Sprintf("servers:ip:%s", server.IP),
		fmt.Sprintf("servers:country_id:%d", server.CountryID),
	}
	for _, key := range cacheKeys {
		if err := sr.cache.Del(context.Background(), key).Err(); err != nil {
			sr.log.Error(constants.ErrDeleteDataFromCache, err)
			return err
		}
	}

	return nil
}
