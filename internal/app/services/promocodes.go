package services

import (
	"database/sql"
	"errors"
	"log/slog"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
)

type Promocodes struct {
	log *logger.Logger
	pr  *repository.Promocodes
}

func NewPromocodes(log *logger.Logger, pr *repository.Promocodes) *Promocodes {
	return &Promocodes{
		log: log,
		pr:  pr,
	}
}

func (ps *Promocodes) IsWork(code string, onlyNewUsers bool) bool {
	if code == "" {
		return false
	}

	data, err := ps.pr.GetByCode(code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}
		ps.log.Error("Failed to get promocode by code", err, slog.String("code", code))
		return false
	}

	if data.OnlyNewUsers != onlyNewUsers || !data.IsActive || data.CurrentActivations > *data.TotalActivations {
		return false
	}

	return true
}

func (ps *Promocodes) GetAll(includeInactive bool) ([]models.Promocode, error) {
	return ps.pr.GetAll(includeInactive)
}

func (ps *Promocodes) GetByCode(code string) (models.Promocode, error) {
	if code == "" {
		return models.Promocode{}, errors.New("code is empty")
	}

	return ps.pr.GetByCode(code)
}

func (ps *Promocodes) Add(promocode models.Promocode) error {
	if promocode.Discount == 0 || promocode.CurrentActivations == 0 || promocode.Code == "" {
		return errors.New("discount, currentActivations, or code is empty")
	}

	return ps.pr.Add(promocode)
}

func (ps *Promocodes) Update(code string, promocode models.Promocode) error {
	if code == "" || promocode == (models.Promocode{}) {
		return errors.New("code or promocode is empty")
	}

	return ps.pr.Update(code, promocode)
}

func (ps *Promocodes) UpdateOnlyNewUsers(code string, onlyNew bool) error {
	if code == "" {
		return errors.New("code is empty")
	}

	return ps.pr.UpdateOnlyNewUsers(code, onlyNew)
}

func (ps *Promocodes) UpdateIsActive(code string, isActive bool) error {
	if code == "" {
		return errors.New("code is empty")
	}

	return ps.pr.UpdateIsActive(code, isActive)
}

func (ps *Promocodes) Delete(code string) error {
	if code == "" {
		return errors.New("code is empty")
	}

	return ps.pr.Delete(code)
}
