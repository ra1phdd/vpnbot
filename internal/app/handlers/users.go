package handlers

import "nsvpn/pkg/logger"

type Users struct {
	log *logger.Logger
}

func NewUsers(log *logger.Logger) *Users {
	return &Users{
		log: log,
	}
}
