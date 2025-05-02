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
	"time"
)

type Subscriptions struct {
	log *logger.Logger
	sr  *repository.Subscriptions

	Plans  *SubscriptionsPlans
	Prices *SubscriptionsPrices
}

func NewSubscriptions(log *logger.Logger, sr *repository.Subscriptions) *Subscriptions {
	return &Subscriptions{
		log: log,
		sr:  sr,
		Plans: &SubscriptionsPlans{
			log: log,
			sr:  sr,
		},
		Prices: &SubscriptionsPrices{
			log: log,
			sr:  sr,
		},
	}
}

func (ss *Subscriptions) GetAllActive() (subscriptions []*models.Subscription, err error) {
	return ss.sr.GetAllActive()
}

func (ss *Subscriptions) GetAllByUserID(userID int64) (subscriptions []*models.Subscription, err error) {
	if userID == 0 {
		return nil, constants.ErrEmptyFields
	}

	return ss.sr.GetAllByUserID(userID)
}

func (ss *Subscriptions) GetLastByUserID(userID int64, isActive bool) (subscription *models.Subscription, err error) {
	if userID == 0 {
		return nil, constants.ErrEmptyFields
	}

	return ss.sr.GetLastByUserID(userID, isActive)
}

func (ss *Subscriptions) Add(subscription *models.Subscription) (int, error) {
	if subscription.UserID == 0 || (subscription.EndDate.Before(subscription.StartDate) && !subscription.EndDate.IsZero()) {
		return 0, constants.ErrEmptyFields
	}

	return ss.sr.Add(subscription)
}

func (ss *Subscriptions) UpdateEndDate(subID int, userID int64, endDate time.Time) error {
	if subID == 0 || userID == 0 || endDate.Before(time.Now()) {
		return constants.ErrEmptyFields
	}

	return ss.sr.UpdateEndDate(subID, userID, endDate)
}

func (ss *Subscriptions) UpdateIsActive(subID int, userID int64, isActive bool) error {
	if subID == 0 || userID == 0 {
		return constants.ErrEmptyFields
	}

	return ss.sr.UpdateIsActive(subID, userID, isActive)
}

func (ss *Subscriptions) Delete(subID int, userID int64) error {
	if subID <= 0 || userID == 0 {
		return constants.ErrEmptyFields
	}

	return ss.sr.Delete(subID, userID)
}

func (ss *Subscriptions) IsActive(userID int64, isActive bool) (bool, error) {
	if userID == 0 {
		return false, constants.ErrEmptyFields
	}

	data, err := ss.sr.GetLastByUserID(userID, isActive)
	if err != nil {
		ss.log.Error("Failed to receiving information about a valid user subscription", err)
		return false, err
	}

	if data == nil {
		return false, nil
	}

	if data.EndDate.After(time.Now().UTC()) && data.IsActive {
		return true, nil
	}
	return false, nil
}

func (ss *Subscriptions) GetVlessKey(uuid string, country *models.Country, countryCode string) string {
	return fmt.Sprintf("vless://%s@%s/?type=tcp&security=reality&pbk=%s&fp=random&sni=%s&sid=%s&spx=%s#%s", uuid, net.JoinHostPort(country.Domain, "443"), country.PublicKey, strings.TrimSuffix(country.Dest, ":443"), country.ShortIDs, url.QueryEscape("/"), countryCode)
}
