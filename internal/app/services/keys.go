package services

import (
	"fmt"
	"net"
	"net/url"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
	"strings"
)

type Keys struct {
	log *logger.Logger
	kr  *repository.Keys
}

func NewKeys(log *logger.Logger, kr *repository.Keys) *Keys {
	return &Keys{
		log: log,
		kr:  kr,
	}
}

func (ks *Keys) GetAll(userID int64) (keys []*models.Key, err error) {
	return ks.kr.GetAll(userID)
}

func (ks *Keys) Get(countryID uint, userID int64) (key *models.Key, err error) {
	if countryID == 0 || userID == 0 {
		return nil, constants.ErrEmptyFields
	}

	return ks.kr.Get(countryID, userID)
}

func (ks *Keys) Add(key *models.Key) error {
	if key.UserID == 0 || key.CountryID == 0 || key.UUID == "" {
		return constants.ErrEmptyFields
	}

	return ks.kr.Add(key)
}

func (ks *Keys) Update(countryID uint, userID int64, newKey *models.Key) error {
	if countryID == 0 || userID == 0 || newKey == nil {
		return constants.ErrEmptyFields
	}

	return ks.kr.Update(countryID, userID, newKey)
}

func (ks *Keys) UpdateIsActive(countryID uint, userID int64, isActive bool) error {
	if countryID == 0 || userID == 0 {
		return constants.ErrEmptyFields
	}

	return ks.kr.UpdateIsActive(countryID, userID, isActive)
}

func (ks *Keys) Delete(countryID uint, userID int64) error {
	if countryID == 0 || userID == 0 {
		return constants.ErrEmptyFields
	}

	return ks.kr.Delete(countryID, userID)
}

func (ks *Keys) GetVlessKey(uuid string, country *models.Country, countryCode string) string {
	return fmt.Sprintf("vless://%s@%s/?type=tcp&security=reality&pbk=%s&flow=%s&fp=random&sni=%s&sid=%s&spx=%s#%s", uuid, net.JoinHostPort(country.Domain, "443"), country.PublicKey, country.Flow, strings.TrimSuffix(country.Dest, ":443"), country.ShortIDs, url.QueryEscape("/"), countryCode)
}
