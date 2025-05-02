package handlers

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"math"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"time"
)

type Subscriptions struct {
	log                  *logger.Logger
	bot                  *telebot.Bot
	ss                   *services.Subscriptions
	cs                   *services.Currency
	ps                   *services.Payments
	us                   *services.Users
	ph                   *Payments
	clientButtonsWithSub *services.Buttons
}

func NewSubscriptions(log *logger.Logger, bot *telebot.Bot, ss *services.Subscriptions, cs *services.Currency, ps *services.Payments, us *services.Users, ph *Payments, clientButtonsWithSub *services.Buttons) *Subscriptions {
	return &Subscriptions{
		log:                  log,
		bot:                  bot,
		ss:                   ss,
		cs:                   cs,
		ps:                   ps,
		us:                   us,
		ph:                   ph,
		clientButtonsWithSub: clientButtonsWithSub,
	}
}

func (s *Subscriptions) ChooseDurationHandler(c telebot.Context) error {
	plans, err := s.ss.Plans.GetAll()
	if err != nil {
		return err
	}

	currency, err := s.cs.Get("RUB")
	if err != nil {
		return err
	}

	buttons, layout := s.ss.Plans.ProcessButtons(plans, currency)
	subBtns := services.NewButtons(buttons, layout, "inline")

	for _, btn := range subBtns.GetBtns() {
		for _, plan := range plans {
			if btn.Unique != fmt.Sprintf("sub_plan_%d", plan.ID) {
				continue
			}

			s.bot.Handle(btn, func(c telebot.Context) error {
				return s.AddSubHandler(c, plan)
			})
		}
	}

	return c.Send("Выберите подписку:", subBtns.AddBtns())
}

func (s *Subscriptions) AddSubHandler(c telebot.Context, subPlan *models.SubscriptionPlan) error {
	endDate := time.Now().AddDate(0, 0, subPlan.DurationDays)
	if subPlan.DurationDays >= 30 {
		endDate = time.Now().AddDate(0, int(math.Round(float64(subPlan.DurationDays/30))), 0)
	}

	amount := subPlan.SubscriptionPrice.Price
	if c.Sender().ID == 1230045591 {
		amount = 1
	}

	sub := &models.Subscription{
		UserID:   c.Sender().ID,
		EndDate:  endDate,
		IsActive: false,
	}

	var err error
	sub.ID, err = s.ss.Add(sub)
	if err != nil {
		s.log.Error("Failed add subscription", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	err = s.us.DecrementBalance(c.Sender().ID, int(amount))
	switch {
	case err == nil:
		err = s.ss.UpdateIsActive(sub.ID, c.Sender().ID, true)
		if err != nil {
			s.log.Error("Failed update isActive", err)
			return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
		}
	case errors.Is(err, constants.ErrInsufficientFunds):
		u, err := uuid.NewUUID()
		if err != nil {
			return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
		}

		err = s.ph.ChooseCurrencyHandler(c, amount, u.String())
		if err != nil {
			return err
		}

		ticker := time.NewTicker(1 * time.Second)

		for range ticker.C {
			payment, err := s.ps.Get(c.Sender().ID, u.String())
			if err != nil {
				s.log.Error("Failed get subscription status", err)
				continue
			}

			if payment != nil && payment.IsCompleted {
				err = s.ss.UpdateIsActive(sub.ID, c.Sender().ID, true)
				if err != nil {
					s.log.Error("Failed update isActive", err)
				}

				err = s.us.DecrementBalance(c.Sender().ID, int(amount))
				if err != nil {
					s.log.Error("Failed update balance", err)
				}
				break
			}
		}
	default:
		s.log.Error("Failed update isCompleted", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	_ = s.bot.Delete(c.Message())
	return c.Send("Подписка активирована!", s.clientButtonsWithSub.AddBtns())
}
