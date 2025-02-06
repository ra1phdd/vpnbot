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

type Currency struct{}

func NewCurrency() *Currency {
	return &Currency{}
}

const (
	queryGetCurrency        = `SELECT * FROM currencies WHERE currency_code = $1`
	queryAddCurrency        = `INSERT INTO currencies (currency_code) VALUES ($1) RETURNING id`
	queryUpdateCurrencyCode = `UPDATE currencies SET currency_code = $1 WHERE id = $2`
	queryDeleteCurrency     = `DELETE FROM currencies WHERE currency_code = $1`
)

func (c *Currency) Get(currencyCode string) (currency models.Currency, err error) {
	method := zap.String("method", "repository_Currency_Get")

	cacheKey := fmt.Sprintf("currency:%s", currencyCode)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		logger.Error(constants.ErrGetDataFromCache, method, zap.String("currencyCode", currencyCode), zap.Error(err))
		return models.Currency{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &currency)
		if err != nil {
			logger.Error(constants.ErrUnmarshalDataFromJSON, method, zap.String("currencyCode", currencyCode), zap.Error(err))
			return models.Currency{}, err
		}
		return currency, nil
	}

	err = db.Conn.QueryRowx(queryGetCurrency, currency).Scan(&currency)
	if err != nil {
		logger.Error(constants.ErrGetDataFromDB, method, zap.String("currencyCode", currencyCode), zap.Error(err))
		return models.Currency{}, err
	}

	jsonData, err := json.Marshal(currency)
	if err != nil {
		logger.Error(constants.ErrMarshalDataToJSON, method, zap.String("currencyCode", currencyCode), zap.Error(err))
		return models.Currency{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		logger.Error(constants.ErrSetDataToCache, method, zap.String("currencyCode", currencyCode), zap.Error(err))
		return models.Currency{}, err
	}

	return currency, nil
}

func (c *Currency) Add(currencyCode string) (id int, err error) {
	method := zap.String("method", "repository_Currency_Add")

	err = db.Conn.QueryRow(queryAddCurrency, currencyCode).Scan(&id)
	if err != nil {
		logger.Error(constants.ErrExecQueryFromDB, method, zap.String("currencyCode", currencyCode), zap.Error(err))
		return 0, err
	}

	return id, nil
}

func (c *Currency) Update(currencyCodeOld, currencyCode string) error {
	method := zap.String("method", "repository_Country_Update")

	currencyOld, err := c.Get(currencyCodeOld)
	if err != nil {
		logger.Error("Error executing the Get function", method, zap.String("currencyCodeOld", currencyCodeOld), zap.String("currencyCode", currencyCode), zap.Error(err))
		return err
	}

	if currencyCode != currencyOld.CurrencyCode && currencyCode != "" {
		_, err = db.Conn.Queryx(queryUpdateCurrencyCode, currencyCode, currencyOld.ID)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.String("currencyCodeOld", currencyCodeOld), zap.String("currencyCode", currencyCode), zap.Any("currencyOld", currencyOld), zap.Error(err))
			return err
		}
	}

	return nil
}

func (c *Currency) Delete(currencyCode string) (err error) {
	method := zap.String("method", "repository_Currency_Delete")

	_, err = db.Conn.Exec(queryDeleteCurrency, currencyCode)
	if err != nil {
		logger.Error(constants.ErrExecQueryFromDB, method, zap.String("currencyCode", currencyCode), zap.Error(err))
		return err
	}

	cacheKey := fmt.Sprintf("currency:%s", currencyCode)
	err = cache.Rdb.Del(cache.Ctx, cacheKey).Err()
	if err != nil {
		logger.Error(constants.ErrDeleteDataFromCache, method, zap.String("currencyCode", currencyCode), zap.Error(err))
		return err
	}

	return nil
}
