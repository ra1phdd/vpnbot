package services

import (
	"errors"
	"fmt"
	"net/url"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
	"strings"
	"time"
)

type Subscriptions struct {
	log *logger.Logger
	sr  *repository.Subscriptions
}

func NewSubscriptions(log *logger.Logger, sr *repository.Subscriptions) *Subscriptions {
	return &Subscriptions{
		log: log,
		sr:  sr,
	}
}

func (ss *Subscriptions) IsActive(userId int64, isActive bool) (bool, error) {
	data, err := ss.sr.GetLastByUserId(userId, isActive)
	if err != nil {
		ss.log.Error("Failed to receiving information about a valid user subscription", err)
		return false, err
	}

	if data == (models.Subscription{}) {
		return false, nil
	}

	if data.EndDate.After(time.Now().UTC()) && data.IsActive {
		return true, nil
	}
	return false, nil
}

func (ss *Subscriptions) GetAllByUserId(userId int64) ([]models.Subscription, error) {
	if userId == 0 {
		return nil, errors.New("userId is empty")
	}

	return ss.sr.GetAllByUserId(userId)
}

func (ss *Subscriptions) GetLastByUserId(userId int64, isActive bool) (models.Subscription, error) {
	if userId == 0 {
		return models.Subscription{}, errors.New("userId is empty")
	}

	return ss.sr.GetLastByUserId(userId, isActive)
}

func (ss *Subscriptions) Add(sub models.Subscription) (int, error) {
	if sub.UserID == 0 {
		return 0, errors.New("userId is empty")
	}

	return ss.sr.Add(sub)
}

func (ss *Subscriptions) UpdateEndDate(subId int, userId int64, endDate time.Time) error {
	if subId <= 0 || userId == 0 {
		return errors.New("subId or userId is empty")
	}
	if endDate.Before(time.Now()) {
		return errors.New("endDate is invalid")
	}

	return ss.sr.UpdateEndDate(subId, userId, endDate)
}

func (ss *Subscriptions) UpdateIsActive(subId int, userId int64, isActive bool) error {
	if subId <= 0 || userId == 0 {
		return errors.New("subId or userId is empty")
	}

	return ss.sr.UpdateIsActive(subId, userId, isActive)
}

func (ss *Subscriptions) Delete(subId int, userId int64) error {
	if subId <= 0 || userId == 0 {
		return errors.New("subId or userId is empty")
	}

	return ss.sr.Delete(subId, userId)
}

func (ss *Subscriptions) GetVlessKey(uuid string, server models.Server, countryCode string) string {
	return fmt.Sprintf("vless://%s@%s:443/?type=tcp&security=reality&pbk=%s&fp=random&sni=%s&sid=%s&spx=%s#nsvpn-%s", uuid, server.IP, server.PublicKey, strings.TrimSuffix(server.Dest, ":443"), server.ShortIDs, url.QueryEscape("/"), countryCode)
}
