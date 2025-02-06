package services

import (
	"database/sql"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net/url"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
	"strings"
	"time"
)

type Subscriptions struct {
	sr *repository.Subscriptions
	pr *repository.Payments
}

func NewSubscriptions(sr *repository.Subscriptions) *Subscriptions {
	return &Subscriptions{
		sr: sr,
	}
}

func (s *Subscriptions) IsActive(userId int64) (bool, error) {
	data, err := s.sr.GetLastByUserId(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		logger.Error("Failed to receiving information about a valid user subscription", zap.Error(err))
		return false, err
	}

	if data.EndDate.After(time.Now().UTC()) && data.IsActive {
		return true, nil
	}
	return false, nil
}

func (s *Subscriptions) Add(sub models.Subscription) (int, error) {
	if sub.UserID == 0 {
		return 0, fmt.Errorf("user id is empty")
	}

	return s.sr.Add(sub)
}

func (s *Subscriptions) UpdateIsActive(userId int64, payload string, isActive bool) error {
	payment, err := s.pr.Get(userId, payload)
	if err != nil {
		logger.Error("Failed to get payment", zap.Error(err))
		return err
	}

	return s.sr.UpdateIsActive(payment.SubscriptionID, isActive)
}

func (s *Subscriptions) GetVlessKey(uuid string, server models.Server, countryCode string) string {
	return fmt.Sprintf("vless://%s@%s:443/?type=tcp&security=reality&flow=xtls-rprx-vision&pbk=%s&fp=random&sni=%s&sid=%s&spx=%s#nsvpn-%s", uuid, server.IP, server.PublicKey, strings.TrimSuffix(server.Dest, ":443"), server.ShortIDs, url.QueryEscape("/"), countryCode)
}

func (s *Subscriptions) GetLastByUserId(userId int64) (sub models.Subscription, err error) {
	return s.sr.GetLastByUserId(userId)
}
