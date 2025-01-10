package promocodes

import (
	"fmt"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/db"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s Service) Get() ([]models.PromoCode, error) {
	var promocodes []models.PromoCode

	rows, err := db.Conn.Query(`SELECT * FROM promocodes WHERE is_active = true`)
	if err != nil {
		return []models.PromoCode{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var data models.PromoCode
		err = rows.Scan(&data.ID, &data.Code, &data.Discount, &data.TotalActivations, &data.Activations, &data.IsActive)
		if err != nil {
			return []models.PromoCode{}, err
		}

		promocodes = append(promocodes, data)
	}
	if len(promocodes) == 0 {
		return []models.PromoCode{}, constants.ErrPromoCodeNotFound
	}

	return promocodes, nil
}

func (s Service) GetByCode(code string) (models.PromoCode, error) {
	var promocode models.PromoCode

	rows, err := db.Conn.Query(`SELECT * FROM promocodes WHERE code = $1`, code)
	if err != nil {
		return models.PromoCode{}, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&promocode.ID, &promocode.Code, &promocode.Discount, &promocode.TotalActivations, &promocode.Activations, &promocode.IsActive)
		if err != nil {
			return models.PromoCode{}, err
		}
	}
	if promocode.ID == 0 {
		return models.PromoCode{}, constants.ErrPromoCodeNotFound
	}

	return promocode, nil
}

func (s Service) Add(promocode models.PromoCode) error {
	rows, err := db.Conn.Queryx(`INSERT INTO promocodes (code, discount, total_activations) VALUES ($1, $2, $3)`, promocode.Code, promocode.Discount, promocode.TotalActivations)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

func (s Service) SwitchState(code string, isActive bool) error {
	rows, err := db.Conn.Queryx(`UPDATE promocodes SET is_active = $1 WHERE code = $2`, isActive, code)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

func (s Service) Use(code string) (bool, error) {
	var promocode models.PromoCode

	rows, err := db.Conn.Query(`SELECT discount, total_activations, activations, is_active FROM promocodes WHERE code = $1`, code)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&promocode.Discount, &promocode.TotalActivations, &promocode.Activations, &promocode.IsActive)
		if err != nil {
			return false, err
		}
	}
	if promocode.Discount == 0 {
		return false, constants.ErrPromoCodeNotFound
	}

	if promocode.TotalActivations < promocode.Activations {
		return false, fmt.Errorf("кол-во активаций превышено")
	}
	return true, nil
}
