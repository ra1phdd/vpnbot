package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"nsvpn/internal/app/constants"
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

func (cs *Country) GetAll() (countries []*models.Country, err error) {
	return cs.cr.GetAll()
}

func (cs *Country) Get(code string) (country *models.Country, err error) {
	if code == "" {
		return nil, constants.ErrEmptyFields
	}

	return cs.cr.Get(code)
}

func (cs *Country) Add(country *models.Country) (uint, error) {
	if country.Code == "" || country.Emoji == "" || country.NameRU == "" || country.NameEN == "" ||
		country.Domain == "" || country.PrivateKey == "" || country.PublicKey == "" || country.Flow == "" ||
		country.Dest == "" || country.ServerNames == "" || country.ShortIDs == "" {
		return 0, constants.ErrEmptyFields
	}

	return cs.cr.Add(country)
}

func (cs *Country) Update(code string, newCountry *models.Country) error {
	if code == "" || newCountry == nil {
		return constants.ErrEmptyFields
	}

	return cs.cr.Update(code, newCountry)
}

func (cs *Country) Delete(code string) error {
	if code == "" {
		return constants.ErrEmptyFields
	}

	return cs.cr.Delete(code)
}

func (cs *Country) ProcessButtons(countries []*models.Country) ([]models.ButtonOption, []int) {
	listCountries := make([]models.ButtonOption, 0, len(countries))

	for _, country := range countries {
		listCountries = append(listCountries, models.ButtonOption{
			Value:   country.Code,
			Display: fmt.Sprintf("%s %s", country.Emoji, country.Code),
		})
	}

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

func (cs *Country) GetHash(countries []*models.Country) (string, error) {
	hash := sha256.New()
	enc := json.NewEncoder(hash)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(countries); err != nil {
		return "", fmt.Errorf("json encode error: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
