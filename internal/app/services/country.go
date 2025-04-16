package services

import (
	"errors"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
)

type Country struct {
	log *logger.Logger
	cr  *repository.Country
}

func NewCountry(log *logger.Logger, cr *repository.Country) *Country {
	return &Country{
		log: log,
		cr:  cr,
	}
}

func (cs *Country) ProcessButtons(countries []models.Country) ([]models.ButtonOption, []int) {
	var listCountries []models.ButtonOption

	for _, country := range countries {
		listCountries = append(listCountries, models.ButtonOption{
			Value:   country.CountryCode,
			Display: country.CountryName,
		})
	}

	listCountries = append(listCountries, models.ButtonOption{
		Value:   "back",
		Display: "Назад",
	})

	var groups []int
	remaining := len(listCountries)
	for remaining > 0 {
		if remaining >= 4 {
			groups = append(groups, 4)
			remaining -= 4
		} else {
			groups = append(groups, remaining)
			break
		}
	}

	return listCountries, groups
}

func (cs *Country) GetAll() ([]models.Country, error) {
	return cs.cr.GetAll()
}

func (cs *Country) Get(countryCode string) (models.Country, error) {
	if countryCode == "" {
		return models.Country{}, errors.New("countryCode is empty")
	}

	return cs.cr.Get(countryCode)
}

func (cs *Country) Add(country models.Country) (int, error) {
	if country.CountryCode == "" || country.CountryName == "" {
		return 0, errors.New("countryCode or countryName is empty")
	}

	return cs.cr.Add(country)
}

func (cs *Country) Delete(countryCode string) error {
	if countryCode == "" {
		return errors.New("countryCode is empty")
	}

	return cs.cr.Delete(countryCode)
}
