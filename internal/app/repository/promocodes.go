package repository

import (
	"database/sql"
	"errors"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/db"
)

type Promocodes struct{}

func NewPromocodes() *Promocodes {
	return &Promocodes{}
}

func (p *Promocodes) Get() ([]models.PromoCode, error) {
	var promocodes []models.PromoCode

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
	if len(promocodes) == 0 {
		return []models.PromoCode{}, constants.ErrPromoCodeNotFound
	}

	return promocodes, nil
}

func (p *Promocodes) GetByCode(code string) (models.PromoCode, error) {
	var data models.PromoCode

	err := db.Conn.QueryRowx(`SELECT * FROM promocodes WHERE code = $1`, code).StructScan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.PromoCode{}, constants.ErrServerNotFound
		}
		return models.PromoCode{}, err
	}

	return data, nil
}

func (p *Promocodes) Add(promocode models.PromoCode) error {
	_, err := db.Conn.Exec(`INSERT INTO promocodes (code, discount, total_activations, only_new_users) VALUES ($1, $2, $3, $4)`, promocode.Code, promocode.Discount, promocode.TotalActivations, promocode.OnlyNewUsers)
	return err
}

func (p *Promocodes) SwitchState(code string, isActive bool) error {
	_, err := db.Conn.Exec(`UPDATE promocodes SET is_active = $1 WHERE code = $2`, isActive, code)
	return err
}

func (p *Promocodes) isWork(data models.PromoCode, onlyNewUsers bool) bool {
	if data.OnlyNewUsers != onlyNewUsers || !data.IsActive || data.CurrentActivations > *data.TotalActivations {
		return false
	}

	return true
}
