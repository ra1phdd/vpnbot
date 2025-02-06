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

type Payments struct{}

func NewPayments() *Payments {
	return &Payments{}
}

const (
	queryGetAllPaymentsByUserId       = `SELECT * FROM payments WHERE user_id = $1`
	queryGetPaymentByUserIdAndPayload = `SELECT * FROM payments WHERE user_id = $1 AND payload = $2`
	queryAddPayment                   = `INSERT INTO payments (user_id, amount, currency_id, subscription_id, payload, is_completed) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	queryUpdateAmountPayment          = `UPDATE payments SET amount = $1 WHERE id = $2`
	queryUpdateCurrencyIdPayment      = `UPDATE payments SET currency_id = $1 WHERE id = $2`
	queryUpdateDatePayment            = `UPDATE payments SET date = $1 WHERE id = $2`
	queryUpdateSubIdPayment           = `UPDATE payments SET subscription_id = $1 WHERE id = $2`
	queryUpdateIsCompletedPayment     = `UPDATE payments SET is_completed = $1 WHERE user_id = $2 AND payload = $3`
	queryDeletePayment                = `DELETE FROM payments WHERE user_id = $1 AND server_id = $2`
)

func (p *Payments) GetAllByUserId(userId int64) (payments []models.Payment, err error) {
	method := zap.String("method", "repository_Payments_GetAllByUserId")

	cacheKey := fmt.Sprintf("payment:user_id:%d", userId)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		logger.Error(constants.ErrGetDataFromCache, method, zap.Int64("userId", userId), zap.Error(err))
		return nil, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &payments)
		if err != nil {
			logger.Error(constants.ErrUnmarshalDataFromJSON, method, zap.Int64("userId", userId), zap.Error(err))
			return nil, err
		}
		return payments, nil
	}

	rows, err := db.Conn.Queryx(queryGetAllPaymentsByUserId, userId)
	if err != nil {
		logger.Error(constants.ErrGetDataFromDB, method, zap.Int64("userId", userId), zap.Error(err))
		return nil, err
	}

	for rows.Next() {
		var data models.Payment
		if err := rows.StructScan(&data); err != nil {
			logger.Error(constants.ErrRowsScanFromDB, method, zap.Int64("userId", userId), zap.Error(err))
			return nil, err
		}

		payments = append(payments, data)
	}

	jsonData, err := json.Marshal(payments)
	if err != nil {
		logger.Error(constants.ErrMarshalDataToJSON, method, zap.Int64("userId", userId), zap.Error(err))
		return nil, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		logger.Error(constants.ErrSetDataToCache, method, zap.Int64("userId", userId), zap.Error(err))
		return nil, err
	}

	return payments, nil
}

func (p *Payments) Get(userId int64, payload string) (payment models.Payment, err error) {
	method := zap.String("method", "repository_Payments_Get")

	cacheKey := fmt.Sprintf("payment:user_id:%d:payload:%s", userId, payload)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		logger.Error(constants.ErrGetDataFromCache, method, zap.Int64("userId", userId), zap.String("payload", payload), zap.Error(err))
		return models.Payment{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &payment)
		if err != nil {
			logger.Error(constants.ErrUnmarshalDataFromJSON, method, zap.Int64("userId", userId), zap.String("payload", payload), zap.Error(err))
			return models.Payment{}, err
		}
		return payment, nil
	}

	err = db.Conn.QueryRowx(queryGetPaymentByUserIdAndPayload, userId, payload).StructScan(&payment)
	if err != nil {
		logger.Error(constants.ErrGetDataFromDB, method, zap.Int64("userId", userId), zap.String("payload", payload), zap.Error(err))
		return models.Payment{}, err
	}

	jsonData, err := json.Marshal(payment)
	if err != nil {
		logger.Error(constants.ErrMarshalDataToJSON, method, zap.Int64("userId", userId), zap.String("payload", payload), zap.Error(err))
		return models.Payment{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		logger.Error(constants.ErrSetDataToCache, method, zap.Int64("userId", userId), zap.String("payload", payload), zap.Error(err))
		return models.Payment{}, err
	}

	return payment, nil
}

func (p *Payments) Add(data models.Payment) (err error) {
	method := zap.String("method", "repository_Payments_Add")

	_, err = db.Conn.Exec(queryAddPayment, data.UserID, data.Amount, data.CurrencyID, data.SubscriptionID, data.Payload, data.IsCompleted)
	if err != nil {
		logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("data", data), zap.Error(err))
		return err
	}
	return nil
}

func (p *Payments) Update(payment models.Payment) error {
	method := zap.String("method", "repository_Payments_Update")

	paymentOld, err := p.Get(payment.UserID, payment.Payload)
	if err != nil {
		logger.Error("Error executing the Get function", method, zap.Any("payment", payment), zap.Error(err))
		return err
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		logger.Error(constants.ErrBeginTx, method, zap.Error(err))
		return err
	}
	defer tx.Rollback()

	if payment.Amount != paymentOld.Amount && payment.Amount > 0 {
		_, err = tx.Exec(queryUpdateAmountPayment, payment.Amount, paymentOld.ID)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("payment", payment), zap.Any("paymentOld", paymentOld), zap.Error(err))
			return err
		}
	}

	if payment.CurrencyID != paymentOld.CurrencyID && payment.CurrencyID != 0 {
		_, err = tx.Exec(queryUpdateCurrencyIdPayment, payment.CurrencyID, paymentOld.ID)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("payment", payment), zap.Any("paymentOld", paymentOld), zap.Error(err))
			return err
		}
	}

	if payment.Date != paymentOld.Date && !payment.Date.After(time.Now()) {
		_, err = tx.Exec(queryUpdateDatePayment, payment.Date, paymentOld.ID)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("payment", payment), zap.Any("paymentOld", paymentOld), zap.Error(err))
			return err
		}
	}

	if payment.SubscriptionID != paymentOld.SubscriptionID && payment.SubscriptionID != 0 {
		_, err = tx.Exec(queryUpdateSubIdPayment, payment.SubscriptionID, paymentOld.ID)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("payment", payment), zap.Any("paymentOld", paymentOld), zap.Error(err))
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Error(constants.ErrCommitTx, method, zap.Error(err))
		return err
	}

	cacheKey := fmt.Sprintf("payment:user_id:%d:payload:%s", payment.UserID, payment.Payload)
	err = cache.Rdb.Del(cache.Ctx, cacheKey).Err()
	if err != nil {
		logger.Error(constants.ErrDeleteDataFromCache, method, zap.Any("payment", payment), zap.Any("paymentOld", paymentOld), zap.Error(err))
		return err
	}

	return nil
}

func (p *Payments) UpdateIsCompleted(userId int64, payload string, isCompleted bool) (err error) {
	method := zap.String("method", "repository_Payments_UpdateIsCompleted")

	_, err = db.Conn.Exec(queryUpdateIsCompletedPayment, isCompleted, userId, payload)
	if err != nil {
		logger.Error(constants.ErrExecQueryFromDB, method, zap.Int64("userId", userId), zap.String("payload", payload), zap.Bool("isCompleted", isCompleted), zap.Error(err))
		return err
	}

	cacheKey := fmt.Sprintf("payment:user_id:%d:payload:%s", userId, payload)
	err = cache.Rdb.Del(cache.Ctx, cacheKey).Err()
	if err != nil {
		logger.Error(constants.ErrDeleteDataFromCache, method, zap.Int64("userId", userId), zap.String("payload", payload), zap.Bool("isCompleted", isCompleted), zap.Error(err))
		return err
	}

	return nil
}

func (p *Payments) Delete(userId int64, payload string) (err error) {
	method := zap.String("method", "repository_Payments_Delete")

	_, err = db.Conn.Exec(queryDeletePayment, userId, payload)
	if err != nil {
		logger.Error(constants.ErrExecQueryFromDB, method, zap.Int64("userId", userId), zap.String("payload", payload), zap.Error(err))
		return err
	}

	cacheKey := fmt.Sprintf("payment:user_id:%d:payload:%s", userId, payload)
	err = cache.Rdb.Del(cache.Ctx, cacheKey).Err()
	if err != nil {
		logger.Error(constants.ErrDeleteDataFromCache, method, zap.Int64("userId", userId), zap.String("payload", payload), zap.Error(err))
		return err
	}

	return nil
}
