package handlers

import (
	"nsvpn/internal/app/models"
)

type User interface {
	GetById(id int64) (models.User, error)
	Add(user models.User) error
	Update(user models.User) error
	Delete(id int64) error
}

type Endpoint struct {
	User User
}
