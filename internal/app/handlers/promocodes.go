package handlers

import (
	"fmt"
	"gopkg.in/telebot.v3"
	"nsvpn/internal/app/models"
)

type Promocodes interface {
	Get() ([]models.PromoCode, error)
	GetByCode(code string) (models.PromoCode, error)
	Add(promocode models.PromoCode) error
	SwitchState(code string, isActive bool) error
	Use(code string) (bool, error)
}

type Endpoint struct {
	Promocodes Promocodes
}

func (e *Endpoint) GetPromocodesHandler(c telebot.Context) error {
	if c.Sender().ID != 1230045591 {
		return nil
	}

	args := c.Args()
	var promocodes []models.PromoCode

	// /get code
	if len(args) == 1 {
		item, err := e.Promocodes.GetByCode(args[0])
		if err != nil {
			return err
		}

		promocodes = append(promocodes, item)
	} else {
		var err error
		promocodes, err = e.Promocodes.Get()
		if err != nil {
			return err
		}
	}

	msg := "Список промокодов:\n"
	for _, promocode := range promocodes {
		msg += fmt.Sprintf("- %s на %d%% (%d/%d)", promocode.Code, promocode.Discount, promocode.Activations, promocode.TotalActivations)
	}

	return c.Send(msg)
}
