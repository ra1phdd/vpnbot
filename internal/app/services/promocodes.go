package services

import (
	"log/slog"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
)

type Promocodes struct {
	log *logger.Logger
	pr  *repository.Promocodes

	Activations *PromocodesActivations
}

func NewPromocodes(log *logger.Logger, pr *repository.Promocodes) *Promocodes {
	return &Promocodes{
		log: log,
		pr:  pr,
		Activations: &PromocodesActivations{
			log: log,
			pr:  pr,
		},
	}
}

func (ps *Promocodes) GetAll() (promocodes []*models.Promocode, err error) {
	return ps.pr.GetAll()
}

func (ps *Promocodes) GetByID(id uint) (promocode *models.Promocode, err error) {
	if id == 0 {
		return nil, constants.ErrEmptyFields
	}

	return ps.pr.GetByID(id)
}

func (ps *Promocodes) Get(code string) (promocode *models.Promocode, err error) {
	if code == "" {
		return nil, constants.ErrEmptyFields
	}

	return ps.pr.Get(code)
}

func (ps *Promocodes) Add(promocode *models.Promocode) error {
	if promocode.Code == "" || promocode.Discount == 0 || promocode.CurrentActivations == 0 {
		return constants.ErrEmptyFields
	}

	return ps.pr.Add(promocode)
}

func (ps *Promocodes) UpdateByID(id uint, newPromocode *models.Promocode) error {
	if id == 0 || newPromocode == nil {
		return constants.ErrEmptyFields
	}

	return ps.pr.UpdateByID(id, newPromocode)
}

func (ps *Promocodes) Update(code string, newPromocode *models.Promocode) error {
	if code == "" || newPromocode == nil {
		return constants.ErrEmptyFields
	}

	return ps.pr.Update(code, newPromocode)
}

func (ps *Promocodes) UpdateOnlyNewUsers(code string, onlyNewUsers bool) error {
	if code == "" {
		return constants.ErrEmptyFields
	}

	return ps.pr.UpdateOnlyNewUsers(code, onlyNewUsers)
}

func (ps *Promocodes) UpdateIsActive(code string, isActive bool) error {
	if code == "" {
		return constants.ErrEmptyFields
	}

	return ps.pr.UpdateIsActive(code, isActive)
}

func (ps *Promocodes) IncrementActivationsByID(id uint) error {
	if id == 0 {
		return constants.ErrEmptyFields
	}

	return ps.pr.IncrementActivationsByID(id)
}

func (ps *Promocodes) Delete(code string) error {
	if code == "" {
		return constants.ErrEmptyFields
	}

	return ps.pr.Delete(code)
}

func (ps *Promocodes) IsWork(code string, isNewUsers bool) bool {
	if code == "" {
		return false
	}

	data, err := ps.pr.Get(code)
	if err != nil {
		ps.log.Error("Failed to get promocode by code", err, slog.String("code", code))
		return false
	}

	if data.OnlyNewUsers != isNewUsers || !data.IsActive || data.CurrentActivations > data.TotalActivations {
		return false
	}

	return true
}
