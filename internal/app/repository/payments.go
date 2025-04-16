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

type Payments struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *redis.Client
}

func NewPayments(log *logger.Logger, db *gorm.DB, cache *redis.Client) *Payments {
	return &Payments{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (pr *Payments) GetAll(userId int64) ([]models.Payment, error) {
	cacheKey := fmt.Sprintf("payment:user_id:%d", userId)
	cacheValue, err := pr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		pr.log.Error(constants.ErrGetDataFromCache, err)
		return nil, err
	}

	var payments []models.Payment
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &payments); err != nil {
			pr.log.Error(constants.ErrUnmarshalDataFromJSON, err)
			return nil, err
		}
		return payments, nil
	}

	if err := pr.db.Where("user_id = ?", userId).Find(&payments).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		pr.log.Error(constants.ErrGetDataFromDB, err)
		return nil, err
	}

	jsonData, err := json.Marshal(payments)
	if err != nil {
		pr.log.Error(constants.ErrMarshalDataToJSON, err)
		return nil, err
	}

	if err := pr.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		pr.log.Error(constants.ErrSetDataToCache, err)
		return nil, err
	}

	return payments, nil
}

func (pr *Payments) Get(userId int64, payload string) (models.Payment, error) {
	cacheKey := fmt.Sprintf("payment:user_id:%d:payload:%s", userId, payload)
	cacheValue, err := pr.cache.Get(context.Background(), cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		pr.log.Error(constants.ErrGetDataFromCache, err)
		return models.Payment{}, err
	}

	var payment models.Payment
	if cacheValue != "" {
		if err := json.Unmarshal([]byte(cacheValue), &payment); err != nil {
			pr.log.Error(constants.ErrUnmarshalDataFromJSON, err)
			return models.Payment{}, err
		}
		return payment, nil
	}

	if err := pr.db.Where("user_id = ? AND payload = ?", userId, payload).First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Payment{}, nil
		}

		pr.log.Error(constants.ErrGetDataFromDB, err)
		return models.Payment{}, err
	}

	jsonData, err := json.Marshal(payment)
	if err != nil {
		pr.log.Error(constants.ErrMarshalDataToJSON, err)
		return models.Payment{}, err
	}

	if err := pr.cache.Set(context.Background(), cacheKey, jsonData, 15*time.Minute).Err(); err != nil {
		pr.log.Error(constants.ErrSetDataToCache, err)
		return models.Payment{}, err
	}

	return payment, nil
}

func (pr *Payments) Add(data models.Payment) error {
	if err := pr.db.Create(&data).Error; err != nil {
		pr.log.Error(constants.ErrExecQueryFromDB, err)
		return err
	}
	return nil
}

func (pr *Payments) Update(userId int64, payload string, payment models.Payment) error {
	paymentOld, err := pr.Get(userId, payload)
	if err != nil {
		pr.log.Error("Error executing the Get function", err)
		return err
	}

	tx := pr.db.Begin()
	if tx.Error != nil {
		pr.log.Error(constants.ErrBeginTx, tx.Error)
		return tx.Error
	}

	if payment.Amount > 0 && payment.Amount != paymentOld.Amount {
		if err := tx.Model(&paymentOld).Where("user_id = ? AND payload = ?", userId, payload).Update("amount", payment.Amount).Error; err != nil {
			tx.Rollback()
			pr.log.Error(constants.ErrExecQueryFromDB, err)
			return err
		}
	}

	if payment.CurrencyID != 0 && payment.CurrencyID != paymentOld.CurrencyID {
		if err := tx.Model(&paymentOld).Where("user_id = ? AND payload = ?", userId, payload).Update("currency_id", payment.CurrencyID).Error; err != nil {
			tx.Rollback()
			pr.log.Error(constants.ErrExecQueryFromDB, err)
			return err
		}
	}

	if !payment.Date.IsZero() && payment.Date != paymentOld.Date && !payment.Date.After(time.Now()) {
		if err := tx.Model(&paymentOld).Where("user_id = ? AND payload = ?", userId, payload).Update("date", payment.Date).Error; err != nil {
			tx.Rollback()
			pr.log.Error(constants.ErrExecQueryFromDB, err)
			return err
		}
	}

	if payment.SubscriptionID != 0 && payment.SubscriptionID != paymentOld.SubscriptionID {
		if err := tx.Model(&paymentOld).Where("user_id = ? AND payload = ?", userId, payload).Update("subscription_id", payment.SubscriptionID).Error; err != nil {
			tx.Rollback()
			pr.log.Error(constants.ErrExecQueryFromDB, err)
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		pr.log.Error(constants.ErrCommitTx, err)
		return err
	}

	cacheKey := fmt.Sprintf("payment:user_id:%d:payload:%s", paymentOld.UserID, paymentOld.Payload)
	if err := pr.cache.Del(context.Background(), cacheKey).Err(); err != nil {
		pr.log.Error(constants.ErrDeleteDataFromCache, err)
		return err
	}

	return nil
}

func (pr *Payments) UpdateIsCompleted(userId int64, payload string, isCompleted bool) error {
	if err := pr.db.Model(&models.Payment{}).
		Where("user_id = ? AND payload = ?", userId, payload).
		Update("is_completed", isCompleted).Error; err != nil {
		pr.log.Error(constants.ErrExecQueryFromDB, err)
		return err
	}

	cacheKey := fmt.Sprintf("payment:user_id:%d:payload:%s", userId, payload)
	if err := pr.cache.Del(context.Background(), cacheKey).Err(); err != nil {
		pr.log.Error(constants.ErrDeleteDataFromCache, err)
		return err
	}

	return nil
}

func (pr *Payments) Delete(userId int64, payload string) error {
	if err := pr.db.Where("user_id = ? AND payload = ?", userId, payload).Delete(&models.Payment{}).Error; err != nil {
		pr.log.Error(constants.ErrExecQueryFromDB, err)
		return err
	}

	cacheKey := fmt.Sprintf("payment:user_id:%d:payload:%s", userId, payload)
	if err := pr.cache.Del(context.Background(), cacheKey).Err(); err != nil {
		pr.log.Error(constants.ErrDeleteDataFromCache, err)
		return err
	}

	return nil
}
