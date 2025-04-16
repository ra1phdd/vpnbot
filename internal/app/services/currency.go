package services

import (
	"errors"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
)

type Currency struct {
	log *logger.Logger
	cr  *repository.Currency
}

func NewCurrency(log *logger.Logger, cr *repository.Currency) *Currency {
	return &Currency{
		log: log,
		cr:  cr,
	}
}

func (cs *Currency) GetAll() ([]models.Currency, error) {
	return cs.cr.GetAll()
}

func (cs *Currency) Get(currencyCode string) (models.Currency, error) {
	if currencyCode == "" {
		return models.Currency{}, errors.New("currencyCode is empty")
	}

	return cs.cr.Get(currencyCode)
}

func (cs *Currency) Add(currency models.Currency) (int, error) {
	if currency.CurrencyCode == "" || currency.CurrencyName == "" {
		return 0, errors.New("id, code or name is empty")
	}

	return currency.ID, nil
}

func (cs *Currency) Delete(currencyCode string) error {
	if currencyCode == "" {
		return errors.New("currencyCode is empty")
	}

	return cs.cr.Delete(currencyCode)
}
