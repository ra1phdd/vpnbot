package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"log/slog"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/logger"
	"time"
)

type Users struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *redis.Client
}

func NewUsers(log *logger.Logger, db *gorm.DB, cache *redis.Client) *Users {
	return &Users{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (ur *Users) GetById(id int64) (models.User, error) {
	cacheKey := fmt.Sprintf("user:%d", id)
	cacheValue, err := ur.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		ur.log.Error("Error getting user from cache", err, slog.Int64("id", id))
		return models.User{}, err
	}

	var user models.User
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &user); err != nil {
			ur.log.Error("Error unmarshaling user from cache", err, slog.Int64("id", id))
			return models.User{}, err
		}
		return user, nil
	}

	if err := ur.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, nil
		}

		ur.log.Error("Error getting user from DB", err, slog.Int64("id", id))
		return models.User{}, err
	}

	jsonData, err := json.Marshal(user)
	if err != nil {
		ur.log.Error("Error marshaling user to JSON", err, slog.Int64("id", id))
		return models.User{}, err
	}

	if err := ur.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		ur.log.Error("Error setting user to cache", err, slog.Int64("id", id))
		return models.User{}, err
	}

	return user, nil
}

func (ur *Users) Add(user models.User) error {
	if err := ur.db.Create(&user).Error; err != nil {
		ur.log.Error("Error inserting user into DB", err, slog.Any("user", user))
		return err
	}
	return nil
}

func (ur *Users) Update(id int64, user models.User) error {
	userOld, err := ur.GetById(id)
	if err != nil {
		ur.log.Error("Error getting existing user", err, slog.Any("user", user))
		return err
	}

	tx := ur.db.Begin()
	if tx.Error != nil {
		ur.log.Error("Error starting transaction", tx.Error)
		return tx.Error
	}

	if user.Username != "" && user.Username != userOld.Username {
		if err := tx.Model(&userOld).Where("id = ?", id).Update("username", user.Username).Error; err != nil {
			tx.Rollback()
			ur.log.Error("Error updating username", err)
			return err
		}
	}

	if user.Firstname != "" && user.Firstname != userOld.Firstname {
		if err := tx.Model(&userOld).Where("id = ?", id).Update("firstname", user.Firstname).Error; err != nil {
			tx.Rollback()
			ur.log.Error("Error updating firstname", err)
			return err
		}
	}

	if user.Lastname != "" && user.Lastname != userOld.Lastname {
		if err := tx.Model(&userOld).Where("id = ?", id).Update("lastname", user.Lastname).Error; err != nil {
			tx.Rollback()
			ur.log.Error("Error updating lastname", err)
			return err
		}
	}

	if user.IsAdmin != userOld.IsAdmin {
		if err := tx.Model(&userOld).Where("id = ?", id).Update("is_admin", user.IsAdmin).Error; err != nil {
			tx.Rollback()
			ur.log.Error("Error updating is_admin", err)
			return err
		}
	}

	if user.IsSign != userOld.IsSign {
		if err := tx.Model(&userOld).Where("id = ?", id).Update("is_sign", user.IsSign).Error; err != nil {
			tx.Rollback()
			ur.log.Error("Error updating is_sign", err)
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		ur.log.Error("Error committing transaction", err)
		return err
	}

	cacheKey := fmt.Sprintf("user:%d", id)
	if err := ur.cache.Del(context.Background(), cacheKey).Err(); err != nil {
		ur.log.Error("Error deleting user from cache", err)
		return err
	}

	return nil
}

func (ur *Users) Delete(id int64) error {
	if err := ur.db.Delete(&models.User{}, id).Error; err != nil {
		ur.log.Error("Error deleting user from DB", err, slog.Int64("id", id))
		return err
	}

	cacheKey := fmt.Sprintf("user:%d", id)
	if err := ur.cache.Del(context.Background(), cacheKey).Err(); err != nil {
		ur.log.Error("Error deleting user from cache", err, slog.Int64("id", id))
		return err
	}

	return nil
}
