package services

import (
	"errors"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
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

func (ks *Keys) Get(serverId int, userId int64) (key models.Key, err error) {
	if serverId == 0 || userId == 0 {
		return models.Key{}, errors.New("serverId or userId is empty")
	}

	return ks.kr.Get(serverId, userId)
}

func (ks *Keys) Add(key models.Key) (err error) {
	if key.UserID == 0 || key.ServerID == 0 || key.UUID == "" {
		return errors.New("userId or serverId or uuid is empty")
	}
	if key.SpeedLimit < 0 || key.TrafficLimit < 0 || key.TrafficUsed < 0 {
		return errors.New("speedLimit or trafficLimit or trafficUsed is invalid")
	}

	return ks.kr.Add(key)
}

func (ks *Keys) Update(serverId int, userId int64, key models.Key) error {
	if serverId == 0 || userId == 0 || key == (models.Key{}) {
		return errors.New("serverId, userId or key is empty")
	}

	return ks.kr.Update(serverId, userId, key)
}

func (ks *Keys) UpdateIsActive(userId int64, serverId int, isActive bool) error {
	if serverId == 0 || userId == 0 {
		return errors.New("serverId or userId is empty")
	}

	return ks.kr.UpdateIsActive(userId, serverId, isActive)
}

func (ks *Keys) Delete(uuid string) error {
	if uuid == "" {
		return errors.New("serverId or userId is empty")
	}

	return ks.kr.Delete(uuid)
}
