package services

import (
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
)

type PromocodesActivations struct {
	log *logger.Logger
	pr  *repository.Promocodes
}

func (ps *PromocodesActivations) GetAll() (activations []*models.PromocodeActivations, err error) {
	return ps.pr.Activations.GetAll()
}

func (ps *PromocodesActivations) GetByUserID(userID int64) (activations []*models.PromocodeActivations, err error) {
	if userID == 0 {
		return nil, constants.ErrEmptyFields
	}

	return ps.pr.Activations.GetByUserID(userID)
}

func (ps *PromocodesActivations) GetByPromocodeID(promocodeID uint) (activations []*models.PromocodeActivations, err error) {
	if promocodeID == 0 {
		return nil, constants.ErrEmptyFields
	}

	return ps.pr.Activations.GetByPromocodeID(promocodeID)
}

func (ps *PromocodesActivations) Get(promocodeID uint, userID int64) (activation *models.PromocodeActivations, err error) {
	if promocodeID == 0 || userID == 0 {
		return nil, constants.ErrEmptyFields
	}

	return ps.pr.Activations.Get(promocodeID, userID)
}

func (ps *PromocodesActivations) Add(activation *models.PromocodeActivations) error {
	if activation.PromocodeID == 0 || activation.UserID == 0 {
		return constants.ErrEmptyFields
	}

	return ps.pr.Activations.Add(activation)
}
