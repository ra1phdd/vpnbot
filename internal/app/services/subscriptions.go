package services

import (
	"errors"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"time"
)

type Subscriptions struct {
	sr *repository.Subscriptions
}

func NewSubscriptions(sr *repository.Subscriptions) *Subscriptions {
	return &Subscriptions{
		sr: sr,
	}
}

func (s *Subscriptions) IsActive(userId int64) (bool, error) {
	data, err := s.sr.GetByUserId(userId)
	if err != nil {
		if errors.Is(err, constants.ErrSubNotFound) {
			return false, nil
		}
		return false, err
	}

	if data.EndDate.Before(time.Now().UTC()) {
		return true, nil
	}
	return false, nil
}

func (s *Subscriptions) Add(sub models.Subscription) (int, error) {
	if sub.UserID == 0 {
		return 0, constants.ErrUserNotFound
	}

	return s.sr.Add(sub)
}

func (s *Subscriptions) UpdateIsActive(userID int64, payload string, isActive bool) error {
	return s.sr.UpdateIsActive(userID, payload, isActive)
}
