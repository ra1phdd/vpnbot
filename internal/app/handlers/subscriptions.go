package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
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
	cs                   *services.Country
	curs                 *services.Currency
	ps                   *services.Payments
	us                   *services.Users
	ph                   *Payments
	clientButtonsWithSub *services.Buttons
}

func NewSubscriptions(log *logger.Logger, bot *telebot.Bot, ss *services.Subscriptions, cs *services.Country, curs *services.Currency, ps *services.Payments, us *services.Users, ph *Payments, clientButtonsWithSub *services.Buttons) *Subscriptions {
	return &Subscriptions{
		log:                  log,
		bot:                  bot,
		ss:                   ss,
		cs:                   cs,
		curs:                 curs,
		ps:                   ps,
		us:                   us,
		ph:                   ph,
		clientButtonsWithSub: clientButtonsWithSub,
	}
}

func (s *Subscriptions) ChooseDurationHandler(c telebot.Context) error {
	btns := getReplyButtons(c)
	plans, err := s.ss.Plans.GetAll()
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	countries, err := s.cs.GetAll()
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	currency, err := s.curs.GetIsBase()
	if err != nil {
		return c.Send(constants.UserError, btns)
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

	msg := "🔥 Вы оформляете подписку на NSVPN.\n\n🌏 Доступные страны:\n"
	for i, country := range countries {
		if i == len(countries)-1 {
			msg += fmt.Sprintf("└ %s %s\n", country.Emoji, country.NameRU)
			break
		}
		msg += fmt.Sprintf("├ %s %s\n", country.Emoji, country.NameRU)
	}

	return c.Send(msg, subBtns.AddBtns())
}

func (s *Subscriptions) AddSubHandler(c telebot.Context, subPlan *models.SubscriptionPlan) error {
	btns := getReplyButtons(c)
	startTime := time.Now()

	currentSub, subOk := c.Get("sub").(*models.Subscription)
	if subOk && currentSub.IsActive && currentSub.EndDate.After(time.Now()) {
		startTime = currentSub.EndDate
	}

	sub, err := s.createOrUpdateSubscription(c.Sender().ID, currentSub, subPlan, startTime)
	if err != nil {
		s.log.Error("Subscription error", err)
		return c.Send(constants.UserError, btns)
	}

	amount := subPlan.SubscriptionPrice.Price
	if err := s.handlePayment(c, sub, subPlan, amount); err != nil {
		if errors.Is(err, constants.ErrPaymentTimeExpired) {
			return c.Send("❌ Время оплаты истекло", btns)
		}

		s.log.Error("Payment error", err)
		return c.Send(constants.UserError, btns)
	}

	if err := s.bot.Delete(c.Message()); err != nil {
		s.log.Warn("Failed to delete message", err)
	}
	return c.Send("✅ Подписка активирована!", s.clientButtonsWithSub.AddBtns())
}

func (s *Subscriptions) createOrUpdateSubscription(userID int64, currentSub *models.Subscription, subPlan *models.SubscriptionPlan, startTime time.Time) (*models.Subscription, error) {
	endDate := startTime.AddDate(0, int(subPlan.DurationDays/30), int(subPlan.DurationDays%30))
	sub := &models.Subscription{
		UserID:   userID,
		IsActive: false,
		EndDate:  endDate,
	}

	if currentSub != nil && currentSub.IsActive && currentSub.EndDate.After(time.Now()) {
		if err := s.ss.UpdateEndDate(currentSub.ID, userID, endDate); err != nil {
			return nil, fmt.Errorf("updateEndDate error: %w", err)
		}
		sub.ID = currentSub.ID
		return sub, nil
	}

	var err error
	sub.ID, err = s.ss.Add(sub)
	if err != nil {
		return nil, fmt.Errorf("add subscription error: %w", err)
	}

	return sub, nil
}

func (s *Subscriptions) handlePayment(c telebot.Context, sub *models.Subscription, subPlan *models.SubscriptionPlan, amount float64) error {
	if err := s.balancePayment(c, c.Sender().ID, sub, subPlan, amount); err == nil {
		return nil
	} else if !errors.Is(err, constants.ErrInsufficientFunds) {
		s.log.Error("Payment error", err)
		return err
	}

	return s.handleExternalPayment(c, c.Sender().ID, sub, subPlan, amount)
}

func (s *Subscriptions) balancePayment(c telebot.Context, userID int64, sub *models.Subscription, subPlan *models.SubscriptionPlan, amount float64) error {
	if err := s.us.DecrementBalance(userID, amount); err != nil {
		s.log.Error("Payment error", err)
		return err
	}

	err := s.ps.Add(&models.Payment{
		UserID:      c.Sender().ID,
		Amount:      amount,
		Type:        "expense",
		Payload:     uuid.New().String(),
		Note:        "Покупка подписки на " + subPlan.Name,
		IsCompleted: true,
	})
	if err != nil {
		return err
	}

	if err := s.ss.UpdateIsActive(sub.ID, userID, true); err != nil {
		if compErr := s.us.IncrementBalance(userID, amount); compErr != nil {
			s.log.Error("Balance compensation failed", compErr)
		}
		return err
	}

	return nil
}

func (s *Subscriptions) handleExternalPayment(c telebot.Context, userID int64, sub *models.Subscription, subPlan *models.SubscriptionPlan, amount float64) error {
	paymentID, err := uuid.NewUUID()
	if err != nil {
		s.log.Error("UUID generation failed", err)
		return err
	}

	if err := s.ph.ChooseCurrencyHandler(c, amount, paymentID.String(), "Пополнение баланса", true); err != nil {
		return fmt.Errorf("payment init failed: %w", err)
	}

	return s.waitForPaymentConfirmation(c, userID, sub, subPlan, amount, paymentID.String())
}

func (s *Subscriptions) waitForPaymentConfirmation(c telebot.Context, userID int64, sub *models.Subscription, subPlan *models.SubscriptionPlan, amount float64, paymentID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return constants.ErrPaymentTimeExpired
		case <-ticker.C:
			payment, err := s.ps.Get(userID, paymentID)
			if err != nil {
				s.log.Error("Payment status check failed", err)
				continue
			}

			if payment != nil && payment.IsCompleted {
				return s.balancePayment(c, userID, sub, subPlan, amount)
			}
		}
	}
}
