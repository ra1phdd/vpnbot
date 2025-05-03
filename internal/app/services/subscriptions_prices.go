package services

import (
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
)

type SubscriptionsPrices struct {
	log *logger.Logger
	sr  *repository.Subscriptions
}

func (ss *SubscriptionsPrices) GetAll() (price []*models.SubscriptionPrice, err error) {
	return ss.sr.Prices.GetAll()
}

func (ss *SubscriptionsPrices) Get(id uint) (price *models.SubscriptionPrice, err error) {
	if id == 0 {
		return nil, constants.ErrEmptyFields
	}

	return ss.sr.Prices.Get(id)
}

func (ss *SubscriptionsPrices) Add(price *models.SubscriptionPrice) error {
	if price.SubscriptionPlanID == 0 || price.Price == 0 {
		return constants.ErrEmptyFields
	}

	return ss.sr.Prices.Add(price)
}

func (ss *SubscriptionsPrices) UpdatePrice(id uint, price float64) error {
	if id == 0 || price == 0 {
		return constants.ErrEmptyFields
	}

	return ss.sr.Prices.UpdatePrice(id, price)
}

func (ss *SubscriptionsPrices) Delete(id uint) error {
	if id == 0 {
		return constants.ErrEmptyFields
	}

	return ss.sr.Prices.Delete(id)
}
