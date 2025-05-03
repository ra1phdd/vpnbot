package services

import (
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
	"strings"
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

func (cs *Currency) GetAll() (currencies []*models.Currency, err error) {
	return cs.cr.GetAll()
}

func (cs *Currency) Get(code string) (currency *models.Currency, err error) {
	if code == "" {
		return nil, constants.ErrEmptyFields
	}

	return cs.cr.Get(code)
}

func (cs *Currency) GetIsBase() (currency *models.Currency, err error) {
	return cs.cr.GetIsBase()
}

func (cs *Currency) Add(currency *models.Currency) (uint, error) {
	if currency.Code == "" || currency.Symbol == "" || currency.Name == "" || currency.ExchangeRate == 0 {
		return 0, constants.ErrEmptyFields
	}

	return cs.cr.Add(currency)
}

func (cs *Currency) Update(code string, newCurrency *models.Currency) error {
	if code == "" || newCurrency == nil {
		return constants.ErrEmptyFields
	}

	return cs.cr.Update(code, newCurrency)
}

func (cs *Currency) Delete(code string) error {
	if code == "" {
		return constants.ErrEmptyFields
	}

	return cs.cr.Delete(code)
}

func (cs *Currency) ProcessButtons(currencies []*models.Currency) ([]models.ButtonOption, []int) {
	listCurrencies := make([]models.ButtonOption, 0, len(currencies))

	for _, cur := range currencies {
		listCurrencies = append(listCurrencies, models.ButtonOption{
			Value:   "currency_" + strings.ToLower(cur.Code),
			Display: cur.Name,
		})
	}

	var groups []int
	remaining := len(listCurrencies)
	for remaining > 0 {
		if remaining >= 3 {
			groups = append(groups, 3)
			remaining -= 3
		} else {
			groups = append(groups, remaining)
			break
		}
	}

	return listCurrencies, groups
}
