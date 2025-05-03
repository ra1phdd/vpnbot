package repository

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log/slog"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/logger"
	"time"
)

type SubscriptionsPlans struct {
	log   *logger.Logger
	db    *gorm.DB
	cache *cache.Cache
}

func (sr *SubscriptionsPlans) GetAll() (plans []*models.SubscriptionPlan, err error) {
	cacheKey := "subscription_plan:all"
	if err = sr.cache.Get(cacheKey, &plans); err == nil {
		sr.log.Debug("Returning subscription Plans from cache", slog.String("cache_key", cacheKey), slog.Int("count", len(plans)))
		return plans, nil
	}

	if err = sr.db.Preload("SubscriptionPrice").Find(&plans).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sr.cache.Set(cacheKey, plans, 1*time.Hour)
			sr.log.Debug("No subscription Plans found in database")
			return nil, nil
		}

		sr.log.Error("Failed to get data from db", err)
		return nil, err
	}

	sr.cache.Set(cacheKey, plans, 1*time.Hour)
	sr.log.Debug("Returning subscription Plans from db", slog.String("cache_key", cacheKey), slog.Int("count", len(plans)))
	return plans, nil
}

func (sr *SubscriptionsPlans) GetByID(id uint) (plan *models.SubscriptionPlan, err error) {
	cacheKey := fmt.Sprintf("subscription_plan:%d", id)
	if err = sr.cache.Get(cacheKey, &plan); err == nil {
		sr.log.Debug("Returning subscription plan from cache", slog.String("cache_key", cacheKey), slog.Uint64("id", uint64(id)))
		return plan, nil
	}

	if err = sr.db.Preload("SubscriptionPrice").First(&plan, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sr.cache.Set(cacheKey, plan, 1*time.Hour)
			sr.log.Debug("Subscription plan not found in database", slog.Uint64("id", uint64(id)))
			return nil, nil
		}
		sr.log.Error("Failed to get data from db", err, slog.Uint64("id", uint64(id)))
		return nil, err
	}

	sr.cache.Set(cacheKey, plan, 1*time.Hour)
	sr.log.Debug("Returning subscription plan from db", slog.String("cache_key", cacheKey), slog.Uint64("id", uint64(id)))
	return plan, nil
}

func (sr *SubscriptionsPlans) GetByDays(days int) (plan *models.SubscriptionPlan, err error) {
	cacheKey := fmt.Sprintf("subscription_plan:duration_days:%d", days)
	if err = sr.cache.Get(cacheKey, &plan); err == nil {
		sr.log.Debug("Returning subscription plan from cache", slog.String("cache_key", cacheKey), slog.Int("days", days))
		return plan, nil
	}

	if err = sr.db.Preload("SubscriptionPrice").Where("duration_days = ?", days).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sr.cache.Set(cacheKey, plan, 1*time.Hour)
			sr.log.Debug("Subscription plan not found in database", slog.Int("days", days))
			return nil, nil
		}
		sr.log.Error("Failed to get data from db", err, slog.Int("days", days))
		return nil, err
	}

	sr.cache.Set(cacheKey, plan, 1*time.Hour)
	sr.log.Debug("Returning subscription plan from db", slog.String("cache_key", cacheKey), slog.Int("days", days))
	return plan, nil
}

func (sr *SubscriptionsPlans) Add(plan *models.SubscriptionPlan) error {
	if err := sr.db.Create(&plan).Error; err != nil {
		sr.log.Error("Failed to execute query from db", err, slog.Any("plan", plan))
		return err
	}

	sr.cache.Delete("subscription_plan:all")
	sr.log.Debug("Added new subscription plan in db", slog.Any("plan", plan))
	return nil
}

func (sr *SubscriptionsPlans) Update(id uint, newPlan *models.SubscriptionPlan) error {
	plan, err := sr.GetByID(id)
	if err != nil {
		sr.log.Error("Failed to execute query from db", err, slog.Uint64("id", uint64(id)))
		return err
	}

	tx := sr.db.Begin()
	if tx.Error != nil {
		sr.log.Error("Failed to begin transaction", tx.Error, slog.Uint64("id", uint64(id)))
		return tx.Error
	}

	if err = updateField(sr.log, tx, plan, "name", plan.Name, newPlan.Name); err != nil {
		tx.Rollback()
		return err
	}
	if err = updateField(sr.log, tx, plan, "duration_days", plan.DurationDays, newPlan.DurationDays); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		sr.log.Error("Failed to commit transaction", err, slog.Uint64("id", uint64(id)))
		return err
	}

	days := plan.DurationDays
	if newPlan.DurationDays != 0 {
		days = newPlan.DurationDays
	}
	sr.cache.Delete(fmt.Sprintf("subscription_plan:%d", id), "subscription_plan:all", fmt.Sprintf("subscription_plan:duration_days:%d", days))
	sr.log.Debug("Successfully updated subscription newPlan", slog.Uint64("id", uint64(id)))
	return nil
}

func (sr *SubscriptionsPlans) Delete(id uint) error {
	plan, err := sr.GetByID(id)
	if err != nil {
		sr.log.Error("Failed to execute query from db", err, slog.Uint64("id", uint64(id)))
		return err
	}

	if err := sr.db.Delete(&models.SubscriptionPlan{}, id).Error; err != nil {
		sr.log.Error("Failed to delete plan from db", err, slog.Uint64("id", uint64(id)))
		return err
	}

	sr.cache.Delete(fmt.Sprintf("subscription_plan:%d", id), "subscription_plan:all", fmt.Sprintf("subscription_plan:duration_days:%d", plan.DurationDays))
	sr.log.Debug("Deleted subscription plan from db", slog.Uint64("id", uint64(id)))
	return nil
}
