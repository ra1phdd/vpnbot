package handlers

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
)

type Promocodes struct {
	log *logger.Logger
	ps  *services.Promocodes
	us  *services.Users
}

func NewPromocodes(log *logger.Logger, ps *services.Promocodes, us *services.Users) *Promocodes {
	return &Promocodes{
		log: log,
		ps:  ps,
		us:  us,
	}
}

func (p *Promocodes) GetPromocodesHandler(c telebot.Context) error {
	if isAdmin, err := p.us.IsAdmin(c.Sender().ID); !isAdmin {
		if err != nil {
			return err
		}
		return errors.New("вы не являетесь администратором для просмотра данной информации")
	}

	var promocodes []models.Promocode

	msg := "Список промокодов:\n"
	for _, promocode := range promocodes {
		msg += fmt.Sprintf("- %s на %d%% (%d/%d)", promocode.Code, promocode.Discount, promocode.CurrentActivations, promocode.TotalActivations)
	}

	return c.Send(msg)
}
