package services

import (
	"fmt"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
)

type Keys struct {
	kr *repository.Keys
}

func NewKeys(kr *repository.Keys) *Keys {
	return &Keys{
		kr: kr,
	}
}

func (k *Keys) GetByServerId(serverId int, userId int64) (key models.Key, err error) {
	if serverId == 0 {
		return models.Key{}, fmt.Errorf("serverId is empty")
	}

	return k.kr.Get(serverId, userId)
}

func (k *Keys) Add(key models.Key) (err error) {
	switch {
	case key.UserID == 0:
		return fmt.Errorf("userId is empty")
	case key.ServerID == 0:
		return fmt.Errorf("serverId is empty")
	case key.UUID == "":
		return fmt.Errorf("uuid is empty")
	case key.SpeedLimit < 0:
		return fmt.Errorf("speed limit is invalid")
	case key.TrafficLimit < 0:
		return fmt.Errorf("traffic limit is invalid")
	case key.TrafficUsed < 0:
		return fmt.Errorf("traffic used is invalid")
	}

	return k.kr.Add(key)
}
