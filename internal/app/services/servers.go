package services

import (
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
)

type Servers struct {
	sr *repository.Servers
}

func NewServers(sr *repository.Servers) *Servers {
	return &Servers{
		sr: sr,
	}
}

func (s *Servers) GetCountries() (countries []models.Country, err error) {
	return s.sr.GetCountries()
}

func (s *Servers) ProcessCountries(countries []models.Country) ([]models.ButtonOption, []int) {
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
