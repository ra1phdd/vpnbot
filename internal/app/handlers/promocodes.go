package handlers

import (
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
)

type Promocodes struct {
	log *logger.Logger
	ps  *services.Promocodes
}

func NewPromocodes(log *logger.Logger, ps *services.Promocodes) *Promocodes {
	return &Promocodes{
		log: log,
		ps:  ps,
	}
}

//func (p *Promocodes) GetPromocodesHandler(c telebot.Context) error {
//	if isAdmin, err := p.us.IsAdmin(c.Sender().ID); !isAdmin {
//		if err != nil {
//			return err
//		}
//		return fmt.Errorf("вы не являетесь администратором для просмотра данной информации")
//	}
//
//	args := c.Args()
//	var promocodes []models.Promocode
//
//	// /get code
//	if len(args) == 1 {
//		item, err := p.pr.Get(args[0])
//		if err != nil {
//			return err
//		}
//
//		promocodes = append(promocodes, item)
//	} else {
//		var err error
//		promocodes, err = p.pr.Get()
//		if err != nil {
//			return err
//		}
//	}
//
//	msg := "Список промокодов:\n"
//	for _, promocode := range promocodes {
//		msg += fmt.Sprintf("- %s на %d%% (%d/%d)", promocode.Code, promocode.Discount, promocode.Activations, promocode.TotalActivations)
//	}
//
//	return c.Send(msg)
//}
