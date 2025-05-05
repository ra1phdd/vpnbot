package state

import (
	"nsvpn/internal/app/models"
	"time"
)

type KeysState struct {
	UUID    string
	Email   string
	EndDate time.Time
	Country *models.Country
	Servers []*models.Server
}
