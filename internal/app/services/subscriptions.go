package services

import (
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
		return false, err
	}

	if data.EndDate.Before(time.Now().UTC()) {
		return true, nil
	}
	return false, nil
}
