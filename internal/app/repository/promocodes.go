package repository

import (
	"database/sql"
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
)

type Promocodes struct{}

func NewPromocodes() *Promocodes {
	return &Promocodes{}
}

func (p *Promocodes) GetAll() (promocodes []models.PromoCode, err error) {
	cacheKey := "promocodes:all"
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &promocodes)
		if err != nil {
			return nil, err
		}
		return promocodes, nil
	}

	rows, err := db.Conn.Queryx(`SELECT * FROM promocodes WHERE is_active = true`)
	if err != nil {
		return []models.PromoCode{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var data models.PromoCode
		if err := rows.StructScan(&data); err != nil {
			return nil, err
		}

		promocodes = append(promocodes, data)
	}

	jsonData, err := json.Marshal(promocodes)
	if err != nil {
		return nil, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 0).Err()
	if err != nil {
		return nil, err
	}

	if len(promocodes) == 0 {
		return []models.PromoCode{}, constants.ErrPromoCodeNotFound
	}

	return promocodes, nil
}

func (p *Promocodes) GetByCode(code string) (promocode models.PromoCode, err error) {
	cacheKey := fmt.Sprintf("promocodes:%s", code)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return models.PromoCode{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &promocode)
		if err != nil {
			return models.PromoCode{}, err
		}
		return promocode, nil
	}

	err = db.Conn.QueryRowx(`SELECT * FROM promocodes WHERE code = $1`, code).StructScan(&promocode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.PromoCode{}, constants.ErrServerNotFound
		}
		return models.PromoCode{}, err
	}

	jsonData, err := json.Marshal(promocode)
	if err != nil {
		return models.PromoCode{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 0).Err()
	if err != nil {
		return models.PromoCode{}, err
	}

	return promocode, nil
}

func (p *Promocodes) Add(promocode models.PromoCode) error {
	_, err := db.Conn.Exec(`INSERT INTO promocodes (code, discount, total_activations, only_new_users, is_active) VALUES ($1, $2, $3, $4, $5)`, promocode.Code, promocode.Discount, promocode.TotalActivations, promocode.OnlyNewUsers, promocode.IsActive)
	return err
}

func (p *Promocodes) Update(promocode models.PromoCode) error {
	promocodeOld, err := p.GetByCode(promocode.Code)
	if err != nil {
		return err
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if promocode.Discount != promocodeOld.Discount && promocode.Discount > 0 {
		_, err = tx.Exec(`UPDATE promocodes SET discount = $1 WHERE code = $2`, promocode.Discount, promocode.Code)
		if err != nil {
			return err
		}
	}

	if promocode.TotalActivations != promocodeOld.TotalActivations && *promocode.TotalActivations > 0 {
		_, err = tx.Exec(`UPDATE promocodes SET total_activations = $1 WHERE code = $2`, promocode.TotalActivations, promocode.Code)
		if err != nil {
			return err
		}
	}

	if promocode.CurrentActivations != promocodeOld.CurrentActivations && promocode.CurrentActivations > 0 {
		_, err = tx.Exec(`UPDATE promocodes SET total_activations = $1 WHERE code = $2`, promocode.TotalActivations, promocode.Code)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *Promocodes) UpdateOnlyNewUsers(code string, onlyNewUsers bool) error {
	_, err := db.Conn.Exec(`UPDATE promocodes SET only_new_users = $1 WHERE code = $2`, onlyNewUsers, code)
	return err
}

func (p *Promocodes) UpdateIsActive(code string, isActive bool) error {
	_, err := db.Conn.Exec(`UPDATE promocodes SET is_active = $1 WHERE code = $2`, isActive, code)
	return err
}

func (p *Promocodes) IsWork(code string, onlyNewUsers bool) bool {
	data, err := p.GetByCode(code)
	if err != nil {
		logger.Error("Failed to get promocode by code", zap.String("code", code), zap.Error(err))
		return false
	}

	if data.OnlyNewUsers != onlyNewUsers || !data.IsActive || data.CurrentActivations > *data.TotalActivations {
		return false
	}

	return true
}
