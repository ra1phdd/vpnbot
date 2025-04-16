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

type Keys struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *redis.Client
}

func NewKeys(log *logger.Logger, db *gorm.DB, cache *redis.Client) *Keys {
	return &Keys{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (kr *Keys) Get(serverId int, userId int64) (models.Key, error) {
	cacheKey := fmt.Sprintf("key:user_id:%d:server_id:%d", userId, serverId)
	cacheValue, err := kr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		kr.log.Error(constants.ErrGetDataFromCache, err)
		return models.Key{}, err
	}

	var key models.Key
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &key); err != nil {
			kr.log.Error(constants.ErrUnmarshalDataFromJSON, err)
			return models.Key{}, err
		}
		return key, nil
	}

	if err := kr.db.Where("server_id = ? AND user_id = ?", serverId, userId).Order("id DESC").Limit(1).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Key{}, nil
		}

		kr.log.Error(constants.ErrGetDataFromDB, err)
		return models.Key{}, err
	}

	jsonData, err := json.Marshal(key)
	if err != nil {
		kr.log.Error(constants.ErrMarshalDataToJSON, err)
		return models.Key{}, err
	}
	if err := kr.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		kr.log.Error(constants.ErrSetDataToCache, err)
		return models.Key{}, err
	}

	return key, nil
}

func (kr *Keys) Add(data models.Key) error {
	if err := kr.db.Create(&data).Error; err != nil {
		kr.log.Error(constants.ErrExecQueryFromDB, err)
		return err
	}
	return nil
}

func (kr *Keys) Update(serverId int, userId int64, key models.Key) error {
	oldKey, err := kr.Get(serverId, userId)
	if err != nil {
		kr.log.Error("Error executing the Get function", err)
		return err
	}

	tx := kr.db.Begin()
	if tx.Error != nil {
		kr.log.Error(constants.ErrBeginTx, tx.Error)
		return tx.Error
	}

	if key.UUID != "" && key.UUID != oldKey.UUID {
		if err := tx.Model(&oldKey).Where("server_id = ? AND user_id = ?", serverId, userId).Update("uuid", key.UUID).Error; err != nil {
			tx.Rollback()
			kr.log.Error(constants.ErrExecQueryFromDB, err)
			return err
		}
	}
	if key.SpeedLimit != oldKey.SpeedLimit {
		if err := tx.Model(&oldKey).Where("server_id = ? AND user_id = ?", serverId, userId).Update("speed_limit", key.SpeedLimit).Error; err != nil {
			tx.Rollback()
			kr.log.Error(constants.ErrExecQueryFromDB, err)
			return err
		}
	}
	if key.TrafficLimit != oldKey.TrafficLimit {
		if err := tx.Model(&oldKey).Where("server_id = ? AND user_id = ?", serverId, userId).Update("traffic_limit", key.TrafficLimit).Error; err != nil {
			tx.Rollback()
			kr.log.Error(constants.ErrExecQueryFromDB, err)
			return err
		}
	}
	if key.TrafficUsed != oldKey.TrafficUsed {
		if err := tx.Model(&oldKey).Where("server_id = ? AND user_id = ?", serverId, userId).Update("traffic_used", key.TrafficUsed).Error; err != nil {
			tx.Rollback()
			kr.log.Error(constants.ErrExecQueryFromDB, err)
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		kr.log.Error(constants.ErrCommitTx, err)
		return err
	}

	cacheKey := fmt.Sprintf("key:user_id:%d:server_id:%d", oldKey.UserID, oldKey.ServerID)
	if err := kr.cache.Del(context.Background(), cacheKey).Err(); err != nil {
		kr.log.Error(constants.ErrDeleteDataFromCache, err)
		return err
	}

	return nil
}

func (kr *Keys) UpdateIsActive(userId int64, serverId int, isActive bool) error {
	if err := kr.db.Model(&models.Key{}).
		Where("user_id = ? AND server_id = ?", userId, serverId).
		Update("is_active", isActive).Error; err != nil {
		kr.log.Error(constants.ErrExecQueryFromDB, err)
		return err
	}

	cacheKey := fmt.Sprintf("key:user_id:%d:server_id:%d", userId, serverId)
	if err := kr.cache.Del(context.Background(), cacheKey).Err(); err != nil {
		kr.log.Error(constants.ErrDeleteDataFromCache, err)
		return err
	}

	return nil
}

func (kr *Keys) Delete(uuid string) error {
	var key models.Key
	if err := kr.db.Where("uuid = ?", uuid).First(&key).Error; err != nil {
		kr.log.Error(constants.ErrExecQueryFromDB, err)
		return err
	}

	if err := kr.db.Delete(&models.Key{}, key.ID).Error; err != nil {
		kr.log.Error(constants.ErrExecQueryFromDB, err)
		return err
	}

	cacheKey := fmt.Sprintf("key:user_id:%d:server_id:%d", key.UserID, key.ServerID)
	if err := kr.cache.Del(context.Background(), cacheKey).Err(); err != nil {
		kr.log.Error(constants.ErrDeleteDataFromCache, err)
		return err
	}

	return nil
}
