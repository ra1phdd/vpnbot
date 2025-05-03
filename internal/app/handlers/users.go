package handlers

import (
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
)

type Users struct {
	log *logger.Logger
	us  *services.Users
}

func NewUsers(log *logger.Logger, us *services.Users) *Users {
	return &Users{
		log: log,
		us:  us,
	}
}
