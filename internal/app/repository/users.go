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

type Users struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *cache.Cache
}

func NewUsers(log *logger.Logger, db *gorm.DB, cache *cache.Cache) *Users {
	return &Users{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (ur *Users) Get(id int64) (user *models.User, err error) {
	cacheKey := fmt.Sprintf("user:%d", id)
	if err = ur.cache.Get(cacheKey, &user); err == nil {
		ur.log.Debug("Returning user from cache", slog.String("cacheKey", cacheKey), slog.Int64("id", id))
		return user, nil
	}

	if err = ur.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ur.cache.Set(cacheKey, user, 15*time.Minute)
			ur.log.Debug("User not found in database", slog.Int64("id", id))
			return nil, nil
		}

		ur.log.Error("Failed to get user from db", err, slog.Int64("id", id))
		return nil, err
	}

	ur.cache.Set(cacheKey, user, 15*time.Minute)
	ur.log.Debug("Returning user from db", slog.Int64("id", id))
	return user, nil
}

func (ur *Users) CountPartners(id int64) (count int64, err error) {
	cacheKey := fmt.Sprintf("user:%d:count_partners", id)
	if err = ur.cache.Get(cacheKey, &count); err == nil {
		ur.log.Debug("Returning user from cache", slog.String("cacheKey", cacheKey), slog.Int64("id", id))
		return count, nil
	}

	result := ur.db.Model(&models.User{}).Where("partner_id = ?", id).Count(&count)
	if err = result.Error; err != nil {
		ur.log.Error("Failed to count users by partner", err, slog.Int64("id", id))
		return 0, err
	}

	ur.cache.Set(cacheKey, count, 15*time.Minute)
	ur.log.Debug("Returning user from db", slog.Int64("id", id))
	return count, nil
}

func (ur *Users) Add(user *models.User) error {
	if err := ur.db.Create(&user).Error; err != nil {
		ur.log.Error("Failed to execute query from db", err, slog.Any("user", user))
		return err
	}

	if user.PartnerID != 0 {
		ur.cache.Delete(fmt.Sprintf("user:%d:count_partners", user.PartnerID))
	}
	ur.log.Debug("Added new user in db", slog.Any("user", user))
	return nil
}

func (ur *Users) Update(id int64, newUser *models.User) error {
	user, err := ur.Get(id)
	if err != nil {
		ur.log.Error("Error getting existing user", err, slog.Int64("id", id))
		return err
	}

	tx := ur.db.Begin()
	if tx.Error != nil {
		ur.log.Error("Error starting transaction", tx.Error)
		return tx.Error
	}

	if err = updateField(ur.log, tx, user, "username", user.Username, newUser.Username); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(ur.log, tx, user, "firstname", user.Firstname, newUser.Firstname); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(ur.log, tx, user, "lastname", user.Lastname, newUser.Lastname); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(ur.log, tx, user, "balance", user.Balance, newUser.Balance); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		ur.log.Error("Error committing transaction", err)
		return err
	}

	ur.cache.Delete(fmt.Sprintf("user:%d", id))
	ur.log.Debug("Successfully updated user", slog.Int64("id", id))
	return nil
}

func (ur *Users) UpdateIsAdmin(id int64, isAdmin bool) error {
	if err := ur.db.Model(&models.User{}).Where("id = ?", id).Update("is_admin", isAdmin).Error; err != nil {
		ur.log.Error("Failed to update is_admin", err, slog.Int64("id", id), slog.Bool("isAdmin", isAdmin))
		return err
	}

	ur.cache.Delete(fmt.Sprintf("user:%d", id))
	ur.log.Debug("Successfully updated is_admin", slog.Int64("id", id), slog.Bool("isAdmin", isAdmin))
	return nil
}

func (ur *Users) UpdateIsSign(id int64, isSign bool) error {
	if err := ur.db.Model(&models.User{}).Where("id = ?", id).Update("is_sign", isSign).Error; err != nil {
		ur.log.Error("Failed to update is_sign", err, slog.Int64("id", id), slog.Bool("isSign", isSign))
		return err
	}

	ur.cache.Delete(fmt.Sprintf("user:%d", id))
	ur.log.Debug("Successfully updated is_sign", slog.Int64("id", id), slog.Bool("isSign", isSign))
	return nil
}

func (ur *Users) IncrementBalance(id int64, amount float64) error {
	result := ur.db.Model(&models.User{}).
		Where("id = ?", id).
		Update("balance", gorm.Expr("balance + ?", amount))
	if result.Error != nil {
		ur.log.Error("Failed to increment balance", result.Error, slog.Int64("id", id), slog.Float64("amount", amount))
		return result.Error
	}

	if result.RowsAffected == 0 {
		return constants.ErrUserNotFound
	}

	ur.cache.Delete(fmt.Sprintf("user:%d", id))
	ur.log.Debug("Incremented user balance", slog.Int64("id", id), slog.Float64("amount", amount))
	return nil
}

func (ur *Users) DecrementBalance(id int64, amount float64) error {
	result := ur.db.Model(&models.User{}).
		Where("id = ? AND balance >= ?", id, amount).
		Update("balance", gorm.Expr("balance - ?", amount))
	if result.Error != nil {
		ur.log.Error("Failed to decrement balance", result.Error, slog.Int64("id", id), slog.Float64("amount", amount))
		return result.Error
	}

	if result.RowsAffected == 0 {
		var exists bool
		if err := ur.db.Model(&models.User{}).
			Select("count(*) > 0").
			Where("id = ?", id).
			Find(&exists).Error; err != nil {
			return err
		}

		if !exists {
			return constants.ErrUserNotFound
		}
		return constants.ErrInsufficientFunds
	}

	ur.cache.Delete(fmt.Sprintf("user:%d", id))
	ur.log.Debug("Decremented user balance", slog.Int64("id", id), slog.Float64("amount", amount))
	return nil
}

func (ur *Users) Delete(id int64) error {
	if err := ur.db.Delete(&models.User{}, id).Error; err != nil {
		ur.log.Error("Failed to delete user from db", err, slog.Int64("id", id))
		return err
	}

	ur.cache.Delete(fmt.Sprintf("user:%d", id), fmt.Sprintf("user:%d:count_partners", id))
	ur.log.Debug("Deleted user from db", slog.Int64("id", id))
	return nil
}
