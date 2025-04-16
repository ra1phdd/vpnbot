package services

import "nsvpn/pkg/logger"

type Base struct {
	log *logger.Logger
}

func NewBase(log *logger.Logger) *Base {
	return &Base{
		log: log,
	}
}
