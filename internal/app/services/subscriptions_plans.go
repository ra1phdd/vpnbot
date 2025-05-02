package services

import (
	"fmt"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
)

type SubscriptionsPlans struct {
	log *logger.Logger
	sr  *repository.Subscriptions
}

func (ss *SubscriptionsPlans) GetAll() (plans []*models.SubscriptionPlan, err error) {
	return ss.sr.Plans.GetAll()
}

func (ss *SubscriptionsPlans) Get(id int) (plan *models.SubscriptionPlan, err error) {
	if id == 0 {
		return nil, constants.ErrEmptyFields
	}

	return ss.sr.Plans.Get(id)
}

func (ss *SubscriptionsPlans) Add(plan *models.SubscriptionPlan) error {
	if plan.Name == "" || plan.DurationDays == 0 {
		return constants.ErrEmptyFields
	}

	return ss.sr.Plans.Add(plan)
}

func (ss *SubscriptionsPlans) Update(id int, newPlan *models.SubscriptionPlan) error {
	if id == 0 || newPlan == nil {
		return constants.ErrEmptyFields
	}

	return ss.sr.Plans.Update(id, newPlan)
}

func (ss *SubscriptionsPlans) Delete(id int) error {
	if id == 0 {
		return constants.ErrEmptyFields
	}

	return ss.sr.Plans.Delete(id)
}

func (ss *SubscriptionsPlans) ProcessButtons(subPlans []*models.SubscriptionPlan, currency *models.Currency) ([]models.ButtonOption, []int) {
	listSubPlans := make([]models.ButtonOption, 0, len(subPlans))

	for i, plan := range subPlans {
		btn := models.ButtonOption{
			Value: fmt.Sprintf("sub_plan_%d", plan.ID),
		}

		priceText := fmt.Sprintf("%s (%.0f %s", plan.Name, plan.SubscriptionPrice.Price, currency.Symbol)

		if i > 0 {
			previousPlan := subPlans[0]
			dailyPreviousPrice := previousPlan.SubscriptionPrice.Price / float64(previousPlan.DurationDays)
			dailyCurrentPrice := plan.SubscriptionPrice.Price / float64(plan.DurationDays)
			discount := 1 - (dailyCurrentPrice / dailyPreviousPrice)

			if discount > 0 {
				priceText += fmt.Sprintf(" | -%.0f%%)", discount*100)
			} else {
				priceText += ")"
			}
		} else {
			priceText += ")"
		}

		btn.Display = priceText
		listSubPlans = append(listSubPlans, btn)
	}

	var groups []int
	remaining := len(listSubPlans)
	for remaining > 0 {
		groups = append(groups, 1)
		remaining--
	}

	return listSubPlans, groups
}
