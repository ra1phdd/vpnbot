package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/db"
	"nsvpn/pkg/logger"
	"time"
)

type Keys struct{}

func NewKeys() *Keys {
	return &Keys{}
}

const (
	queryGetKeyByServerId      = `SELECT * FROM keys WHERE server_id = $1 AND user_id = $2 ORDER BY id DESC LIMIT 1`
	queryAddKey                = `INSERT INTO keys (user_id, server_id, uuid, speed_limit, traffic_limit, traffic_used, is_active) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	queryUpdateUuidKey         = `UPDATE keys SET uuid = $1 WHERE id = $2`
	queryUpdateSpeedLimitKey   = `UPDATE keys SET speed_limit = $1 WHERE id = $2`
	queryUpdateTrafficLimitKey = `UPDATE keys SET traffic_limit = $1 WHERE id = $2`
	queryUpdateTrafficUsedKey  = `UPDATE keys SET traffic_used = $1 WHERE id = $2`
	queryUpdateIsActiveKey     = `UPDATE keys SET is_active = $1 WHERE user_id = $2 AND server_id = $3`
	queryDeleteKey             = `DELETE FROM keys WHERE uuid = $1 RETURNING user_id, server_id`
)

func (k *Keys) Get(serverId int, userId int64) (key models.Key, err error) {
	method := zap.String("method", "repository_Keys_Get")

	cacheKey := fmt.Sprintf("key:user_id:%d:server_id:%d", userId, serverId)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		logger.Error(constants.ErrGetDataFromCache, method, zap.Int64("userId", userId), zap.Int("serverId", serverId), zap.Error(err))
		return models.Key{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &key)
		if err != nil {
			logger.Error(constants.ErrUnmarshalDataFromJSON, method, zap.Int64("userId", userId), zap.Int("serverId", serverId), zap.Error(err))
			return models.Key{}, err
		}
		return key, nil
	}

	err = db.Conn.QueryRowx(queryGetKeyByServerId, serverId, userId).StructScan(&key)
	if err != nil {
		logger.Error(constants.ErrGetDataFromDB, method, zap.Int64("userId", userId), zap.Int("serverId", serverId), zap.Error(err))
		return models.Key{}, err
	}

	jsonData, err := json.Marshal(key)
	if err != nil {
		logger.Error(constants.ErrMarshalDataToJSON, zap.Int64("userId", userId), method, zap.Int("serverId", serverId), zap.Error(err))
		return models.Key{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		logger.Error(constants.ErrSetDataToCache, zap.Int64("userId", userId), method, zap.Int("serverId", serverId), zap.Error(err))
		return models.Key{}, err
	}

	return key, nil
}

func (k *Keys) Add(data models.Key) (err error) {
	method := zap.String("method", "repository_Keys_Add")

	_, err = db.Conn.Exec(queryAddKey, data.UserID, data.ServerID, data.UUID, data.SpeedLimit, data.TrafficLimit, data.TrafficUsed, data.IsActive)
	if err != nil {
		logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("data", data), zap.Error(err))
		return err
	}
	return nil
}

func (k *Keys) Update(key models.Key) error {
	method := zap.String("method", "repository_Keys_Update")

	keyOld, err := k.Get(key.ServerID, key.UserID)
	if err != nil {
		logger.Error("Error executing the Get function", method, zap.Any("key", key), zap.Error(err))
		return err
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		logger.Error(constants.ErrBeginTx, method, zap.Error(err))
		return err
	}
	defer tx.Rollback()

	if key.UUID != keyOld.UUID && key.UUID != "" {
		_, err = tx.Exec(queryUpdateUuidKey, key.UUID, keyOld.ID)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("key", key), zap.Any("keyOld", keyOld), zap.Error(err))
			return err
		}
	}

	if key.SpeedLimit != keyOld.SpeedLimit {
		_, err = tx.Exec(queryUpdateSpeedLimitKey, key.SpeedLimit, keyOld.ID)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("key", key), zap.Any("keyOld", keyOld), zap.Error(err))
			return err
		}
	}

	if key.TrafficLimit != keyOld.TrafficLimit {
		_, err = tx.Exec(queryUpdateTrafficLimitKey, key.TrafficLimit, keyOld.ID)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("key", key), zap.Any("keyOld", keyOld), zap.Error(err))
			return err
		}
	}

	if key.TrafficUsed != keyOld.TrafficUsed {
		_, err = tx.Exec(queryUpdateTrafficUsedKey, key.TrafficUsed, keyOld.ID)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("key", key), zap.Any("keyOld", keyOld), zap.Error(err))
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Error(constants.ErrCommitTx, method, zap.Error(err))
		return err
	}

	cacheKey := fmt.Sprintf("key:user_id:%d:server_id:%d", key.UserID, key.ServerID)
	err = cache.Rdb.Del(cache.Ctx, cacheKey).Err()
	if err != nil {
		logger.Error(constants.ErrDeleteDataFromCache, method, zap.Any("key", key), zap.Any("keyOld", keyOld), zap.Error(err))
		return err
	}

	return nil
}

func (k *Keys) UpdateIsActive(userId int64, serverId int, isActive bool) (err error) {
	method := zap.String("method", "repository_Keys_UpdateIsActive")

	_, err = db.Conn.Exec(queryUpdateIsActiveKey, isActive, userId, serverId)
	if err != nil {
		logger.Error(constants.ErrExecQueryFromDB, method, zap.Int64("userId", userId), zap.Int("serverId", serverId), zap.Bool("isActive", isActive), zap.Error(err))
		return err
	}

	cacheKey := fmt.Sprintf("key:user_id:%d:server_id:%d", userId, serverId)
	err = cache.Rdb.Del(cache.Ctx, cacheKey).Err()
	if err != nil {
		logger.Error(constants.ErrDeleteDataFromCache, method, zap.Int64("userId", userId), zap.Int("serverId", serverId), zap.Bool("isActive", isActive), zap.Int64("userId", userId), zap.Int("serverId", serverId), zap.Error(err))
		return err
	}

	return nil
}

func (k *Keys) Delete(uuid string) (err error) {
	method := zap.String("method", "repository_Keys_Delete")

	var userId int64
	var serverId int
	err = db.Conn.QueryRow(queryDeleteKey, uuid).Scan(&userId, &serverId)
	if err != nil {
		logger.Error(constants.ErrExecQueryFromDB, method, zap.String("uuid", uuid), zap.Error(err))
		return err
	}

	cacheKey := fmt.Sprintf("key:user_id:%d:server_id:%d", userId, serverId)
	err = cache.Rdb.Del(cache.Ctx, cacheKey).Err()
	if err != nil {
		logger.Error(constants.ErrDeleteDataFromCache, method, zap.Int64("userId", userId), zap.Int("serverId", serverId), zap.Error(err))
		return err
	}

	return nil
}
