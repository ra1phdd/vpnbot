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

type Payments struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *cache.Cache
}

func NewPayments(log *logger.Logger, db *gorm.DB, cache *cache.Cache) *Payments {
	return &Payments{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (pr *Payments) GetAll(userID int64, offset, limit int) (payments []*models.Payment, err error) {
	cacheKey := fmt.Sprintf("payment:user_id:%d:offset:%d:limit:%d", userID, offset, limit)
	if err = pr.cache.Get(cacheKey, &payments); err == nil {
		pr.log.Debug("Returning payments from cache", slog.String("cache_key", cacheKey), slog.Int("count", len(payments)), slog.Int64("user_id", userID))
		return payments, nil
	}

	dbQuery := pr.db.Where("user_id = ? AND is_completed = true", userID)
	if offset > 0 {
		dbQuery = dbQuery.Offset(offset)
	}
	if limit > 0 {
		dbQuery = dbQuery.Limit(limit)
	}

	if err = dbQuery.Find(&payments).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pr.cache.Set(cacheKey, payments, 15*time.Minute)
			pr.log.Debug("No payments found in database", slog.Int64("user_id", userID))
			return nil, nil
		}

		pr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	pr.cache.Set(cacheKey, payments, 15*time.Minute)
	pr.log.Debug("Returning payments from db", slog.String("cache_key", cacheKey), slog.Int("count", len(payments)), slog.Int64("user_id", userID))
	return payments, nil
}

func (pr *Payments) GetPaymentsCount(userID int64) (count int64, err error) {
	cacheKey := fmt.Sprintf("payment:user_id:%d:count", userID)
	if err = pr.cache.Get(cacheKey, &count); err == nil {
		pr.log.Debug("Returning payments from cache", slog.String("cache_key", cacheKey), slog.Int64("count", count), slog.Int64("user_id", userID))
		return count, nil
	}

	err = pr.db.Model(&models.Payment{}).Where("user_id = ? AND is_completed = true", userID).Count(&count).Error
	if err != nil {
		pr.log.Error("Failed to get count from db", err)
		return 0, err
	}

	pr.cache.Set(cacheKey, count, 15*time.Minute)
	pr.log.Debug("Returning count payments from db", slog.String("cache_key", cacheKey), slog.Int64("count", count), slog.Int64("user_id", userID))
	return count, err
}

func (pr *Payments) Get(userID int64, payload string) (payment *models.Payment, err error) {
	cacheKey := fmt.Sprintf("payment:user_id:%d:payload:%s", userID, payload)
	if err = pr.cache.Get(cacheKey, &payment); err == nil {
		pr.log.Debug("Returning payment from cache", slog.String("cache_key", cacheKey), slog.Int64("user_id", userID), slog.String("payload", payload))
		return payment, nil
	}

	if err = pr.db.Where("user_id = ? AND payload = ?", userID, payload).Order("id DESC").First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pr.cache.Set(cacheKey, payment, 15*time.Minute)
			pr.log.Debug("Payment not found in database", slog.Int64("user_id", userID), slog.String("payload", payload))
			return nil, nil
		}

		pr.log.Error("Failed to get payment from db", err, slog.Int64("user_id", userID), slog.String("payload", payload))
		return nil, err
	}

	pr.cache.Set(cacheKey, payment, 15*time.Minute)
	pr.log.Debug("Returning payment from db", slog.Int64("user_id", userID), slog.String("payload", payload))
	return payment, nil
}

func (pr *Payments) Add(payment *models.Payment) error {
	if err := pr.db.Create(&payment).Error; err != nil {
		pr.log.Error("Failed to create payment in db", err, slog.Int64("user_id", payment.UserID), slog.String("payload", payment.Payload))
		return err
	}

	pr.cache.Delete(fmt.Sprintf("payment:user_id:%d:*", payment.UserID), fmt.Sprintf("payment:user_id:%d:count", payment.UserID))
	pr.log.Debug("Added new payment in db", slog.Int64("user_id", payment.UserID), slog.String("payload", payment.Payload))
	return nil
}

func (pr *Payments) Update(userID int64, payload string, newPayment *models.Payment) error {
	payment, err := pr.Get(userID, payload)
	if err != nil {
		pr.log.Error("Failed to get payment for update", err, slog.Int64("user_id", userID), slog.String("payload", payload))
		return err
	}

	tx := pr.db.Begin()
	if tx.Error != nil {
		pr.log.Error("Failed to begin transaction", tx.Error, slog.Int64("user_id", userID), slog.String("payload", payload))
		return tx.Error
	}

	if err = updateField(pr.log, tx, payment, "amount", payment.Amount, newPayment.Amount); err != nil {
		tx.Rollback()
		return err
	}
	if !newPayment.Date.IsZero() && newPayment.Date != payment.Date && !newPayment.Date.After(time.Now()) {
		if err := tx.Model(&payment).Where("user_id = ? AND payload = ?", userID, payload).Update("date", newPayment.Date).Error; err != nil {
			tx.Rollback()
			pr.log.Error("Failed to update date", err, slog.Time("newDate", newPayment.Date))
			return err
		}
	}

	if err = tx.Commit().Error; err != nil {
		pr.log.Error("Failed to commit transaction", err, slog.Int64("user_id", userID), slog.String("payload", payload))
		return err
	}

	pr.cache.Delete(fmt.Sprintf("payment:user_id:%d:*", userID), fmt.Sprintf("payment:user_id:%d:payload:%s", userID, payload))
	pr.log.Debug("Successfully updated payment", slog.Int64("user_id", userID), slog.String("payload", payload))
	return nil
}

func (pr *Payments) UpdateIsCompleted(userID int64, payload string, isCompleted bool) error {
	if err := pr.db.Model(&models.Payment{}).
		Where("user_id = ? AND payload = ?", userID, payload).
		Update("is_completed", isCompleted).Error; err != nil {
		pr.log.Error("Failed to update is_completed", err, slog.Int64("user_id", userID), slog.String("payload", payload))
		return err
	}

	pr.cache.Delete(fmt.Sprintf("payment:user_id:%d:*", userID), fmt.Sprintf("payment:user_id:%d:payload:%s", userID, payload))
	pr.log.Debug("Successfully updated is_completed", slog.Int64("user_id", userID), slog.String("payload", payload), slog.Bool("is_completed", isCompleted))
	return nil
}

func (pr *Payments) Delete(userID int64, payload string) error {
	if err := pr.db.Where("user_id = ? AND payload = ?", userID, payload).Delete(&models.Payment{}).Error; err != nil {
		pr.log.Error("Failed to delete payment", err, slog.Int64("user_id", userID), slog.String("payload", payload))
		return err
	}

	pr.cache.Delete(fmt.Sprintf("payment:user_id:%d:*", userID), fmt.Sprintf("payment:user_id:%d:payload:%s", userID, payload))
	pr.log.Debug("Deleted payment from db", slog.Int64("user_id", userID), slog.String("payload", payload))
	return nil
}
