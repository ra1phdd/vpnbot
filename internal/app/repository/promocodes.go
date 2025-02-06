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

type Promocodes struct{}

func NewPromocodes() *Promocodes {
	return &Promocodes{}
}

const (
	queryGetAllPromocodes                  = `SELECT * FROM promocodes`
	queryGetAllOnlyActivePromocodes        = `SELECT * FROM promocodes WHERE is_active = true`
	queryGetByCodePromocodes               = `SELECT * FROM promocodes WHERE code = $1`
	queryAddPromocode                      = `INSERT INTO promocodes (code, discount, total_activations, current_activations, only_new_users, is_active) VALUES ($1, $2, $3, $4, $5, $6)`
	queryUpdateDiscountPromocode           = `UPDATE promocodes SET discount = $1 WHERE code = $2`
	queryUpdateTotalActivationsPromocode   = `UPDATE promocodes SET total_activations = $1 WHERE code = $2`
	queryUpdateCurrentActivationsPromocode = `UPDATE promocodes SET total_activations = $1 WHERE code = $2`
	queryUpdateOnlyNewUsersPromocode       = `UPDATE promocodes SET only_new_users = $1 WHERE code = $2`
	queryUpdateIsActivePromocode           = `UPDATE promocodes SET is_active = $1 WHERE code = $2`
)

func (p *Promocodes) GetAll(isInactive bool) (promocodes []models.PromoCode, err error) {
	method := zap.String("method", "repository_Promocodes_GetAll")

	var cacheKey string
	if isInactive {
		cacheKey = "promocodes:all"
	} else {
		cacheKey = "promocodes:only_active"
	}

	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		logger.Error(constants.ErrGetDataFromCache, method, zap.Bool("isInactive", isInactive), zap.Error(err))
		return nil, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &promocodes)
		if err != nil {
			logger.Error(constants.ErrUnmarshalDataFromJSON, method, zap.Bool("isInactive", isInactive), zap.Error(err))
			return nil, err
		}
		return promocodes, nil
	}

	query := queryGetAllOnlyActivePromocodes
	if isInactive {
		query = queryGetAllPromocodes
	}

	rows, err := db.Conn.Queryx(query)
	if err != nil {
		logger.Error(constants.ErrGetDataFromDB, method, zap.Bool("isInactive", isInactive), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var promocode models.PromoCode
		if err := rows.StructScan(&promocode); err != nil {
			logger.Error(constants.ErrRowsScanFromDB, method, zap.Bool("isInactive", isInactive), zap.Error(err))
			return nil, err
		}
		promocodes = append(promocodes, promocode)
	}

	jsonData, err := json.Marshal(promocodes)
	if err != nil {
		logger.Error(constants.ErrMarshalDataToJSON, method, zap.Bool("isInactive", isInactive), zap.Error(err))
		return nil, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		logger.Error(constants.ErrSetDataToCache, method, zap.Bool("isInactive", isInactive), zap.Error(err))
		return nil, err
	}

	return promocodes, nil
}

func (p *Promocodes) GetByCode(code string) (promocode models.PromoCode, err error) {
	method := zap.String("method", "repository_Promocodes_GetByCode")

	cacheKey := fmt.Sprintf("promocodes:%s", code)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		logger.Error(constants.ErrGetDataFromCache, method, zap.String("code", code), zap.Error(err))
		return models.PromoCode{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &promocode)
		if err != nil {
			logger.Error(constants.ErrUnmarshalDataFromJSON, method, zap.String("code", code), zap.Error(err))
			return models.PromoCode{}, err
		}
		return promocode, nil
	}

	err = db.Conn.QueryRowx(queryGetByCodePromocodes, code).StructScan(&promocode)
	if err != nil {
		logger.Error(constants.ErrGetDataFromDB, method, zap.String("code", code), zap.Error(err))
		return promocode, err
	}

	jsonData, err := json.Marshal(promocode)
	if err != nil {
		logger.Error(constants.ErrMarshalDataToJSON, method, zap.String("code", code), zap.Error(err))
		return models.PromoCode{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		logger.Error(constants.ErrSetDataToCache, method, zap.String("code", code), zap.Error(err))
		return models.PromoCode{}, err
	}

	return promocode, nil
}

func (p *Promocodes) Add(promocode models.PromoCode) error {
	method := zap.String("method", "repository_Promocodes_Add")

	_, err := db.Conn.Exec(queryAddPromocode, promocode.Code, promocode.Discount, promocode.TotalActivations, promocode.OnlyNewUsers, promocode.IsActive)
	if err != nil {
		logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("data", promocode), zap.Error(err))
		return err
	}
	return nil
}

func (p *Promocodes) Update(promocode models.PromoCode) error {
	method := zap.String("method", "repository_Promocodes_Update")

	promocodeOld, err := p.GetByCode(promocode.Code)
	if err != nil {
		logger.Error("Error executing the GetByCode function", method, zap.Any("promocode", promocode), zap.Error(err))
		return err
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		logger.Error(constants.ErrBeginTx, method, zap.Error(err))
		return err
	}
	defer tx.Rollback()

	if promocode.Discount != promocodeOld.Discount && promocode.Discount > 0 {
		_, err = tx.Exec(queryUpdateDiscountPromocode, promocode.Discount, promocode.Code)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("promocode", promocode), zap.Any("promocodeOld", promocodeOld), zap.Error(err))
			return err
		}
	}

	if promocode.TotalActivations != promocodeOld.TotalActivations && *promocode.TotalActivations > 0 {
		_, err = tx.Exec(queryUpdateTotalActivationsPromocode, promocode.TotalActivations, promocode.Code)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("promocode", promocode), zap.Any("promocodeOld", promocodeOld), zap.Error(err))
			return err
		}
	}

	if promocode.CurrentActivations != promocodeOld.CurrentActivations && promocode.CurrentActivations > 0 {
		_, err = tx.Exec(queryUpdateCurrentActivationsPromocode, promocode.TotalActivations, promocode.Code)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("promocode", promocode), zap.Any("promocodeOld", promocodeOld), zap.Error(err))
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Error(constants.ErrCommitTx, method, zap.Error(err))
		return err
	}

	cacheKey := fmt.Sprintf("promocodes:%s", promocode.Code)
	err = cache.Rdb.Del(cache.Ctx, cacheKey).Err()
	if err != nil {
		logger.Error(constants.ErrDeleteDataFromCache, method, zap.Any("promocode", promocode), zap.Any("promocodeOld", promocodeOld), zap.Error(err))
		return err
	}

	return nil
}

func (p *Promocodes) UpdateOnlyNewUsers(code string, onlyNewUsers bool) error {
	method := zap.String("method", "repository_Promocodes_UpdateOnlyNewUsers")

	_, err := db.Conn.Exec(queryUpdateOnlyNewUsersPromocode, onlyNewUsers, code)
	if err != nil {
		logger.Error(constants.ErrExecQueryFromDB, method, zap.String("code", code), zap.Bool("onlyNewUsers", onlyNewUsers), zap.Error(err))
		return err
	}

	cacheKey := fmt.Sprintf("promocodes:%s", code)
	err = cache.Rdb.Del(cache.Ctx, cacheKey).Err()
	if err != nil {
		logger.Error(constants.ErrDeleteDataFromCache, method, zap.String("code", code), zap.Error(err))
		return err
	}

	return nil
}

func (p *Promocodes) UpdateIsActive(code string, isActive bool) error {
	method := zap.String("method", "repository_Promocodes_UpdateIsActive")

	_, err := db.Conn.Exec(queryUpdateIsActivePromocode, isActive, code)
	if err != nil {
		logger.Error(constants.ErrExecQueryFromDB, method, zap.String("code", code), zap.Bool("isActive", isActive), zap.Error(err))
		return err
	}

	cacheKey := fmt.Sprintf("promocodes:%s", code)
	err = cache.Rdb.Del(cache.Ctx, cacheKey).Err()
	if err != nil {
		logger.Error(constants.ErrDeleteDataFromCache, method, zap.String("code", code), zap.Error(err))
		return err
	}

	return nil
}
