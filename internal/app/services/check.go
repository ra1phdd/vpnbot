package services

import (
	"gopkg.in/telebot.v4"
	"log/slog"
	"nsvpn/internal/app/api"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/logger"
	"time"
)

type Check struct {
	log           *logger.Logger
	bot           *telebot.Bot
	ks            *Keys
	subs          *Subscriptions
	servs         *Servers
	us            *Users
	api           *api.API
	clientButtons *Buttons
}

func NewCheck(log *logger.Logger, bot *telebot.Bot, ks *Keys, subs *Subscriptions, servs *Servers, us *Users, api *api.API, clientButtons *Buttons) *Check {
	return &Check{
		log:           log,
		bot:           bot,
		ks:            ks,
		subs:          subs,
		servs:         servs,
		us:            us,
		api:           api,
		clientButtons: clientButtons,
	}
}

func (c *Check) Run() {
	servers, err := c.servs.GetAll()
	if err != nil {
		c.log.Error("Failed to get all servers", err)
		return
	}

	subscriptions, err := c.subs.GetAllActive()
	if err != nil {
		c.log.Error("Failed to get all subscriptions", err)
		return
	}

	for _, sub := range subscriptions {
		isExpired, msg, opts := c.checkSubscriptionExpiration(sub)
		if !isExpired {
			continue
		}

		if ok := c.tryRenewSubscription(sub, servers); ok {
			msg = "Ваша подписка успешно продлена"
		}

		if _, err := c.bot.Send(&telebot.User{ID: sub.UserID}, msg, opts); err != nil {
			c.log.Error("Failed to send message", err)
		}
	}
}

func (c *Check) checkSubscriptionExpiration(sub *models.Subscription) (bool, string, *telebot.ReplyMarkup) {
	expireTime := time.Until(sub.EndDate)
	now := time.Now()

	isExpired := (expireTime <= 3*time.Hour && expireTime > 2*time.Hour) ||
		(expireTime <= 24*time.Hour && expireTime > 23*time.Hour) ||
		(expireTime <= 72*time.Hour && expireTime > 71*time.Hour) ||
		(expireTime <= 168*time.Hour && expireTime > 167*time.Hour) ||
		(sub.EndDate.Before(now) && !sub.EndDate.IsZero())

	msg := "Ваша подписка истечёт в " + sub.EndDate.Format("2006-01-02 15:04:05")
	var opts *telebot.ReplyMarkup

	if sub.EndDate.Before(now) && !sub.EndDate.IsZero() {
		msg = "Ваша подписка истекла"
		opts = c.clientButtons.AddBtns()
	}

	return isExpired, msg, opts
}

func (c *Check) tryRenewSubscription(sub *models.Subscription, servers []*models.Server) bool {
	plan, err := c.subs.Plans.GetByDays(30)
	if err != nil {
		c.log.Error("Failed to get plan", err)
		c.processServers(sub, servers)
		return false
	}

	if err := c.us.DecrementBalance(sub.UserID, plan.SubscriptionPrice.Price); err != nil {
		c.log.Error("Failed to decrement balance", err)
		c.processServers(sub, servers)
		return false
	}

	newEndDate := time.Now().AddDate(0, 1, 0)
	if err := c.subs.UpdateEndDate(sub.ID, sub.UserID, newEndDate); err != nil {
		c.log.Error("Failed to update subscription", err)

		if rerr := c.us.IncrementBalance(sub.UserID, plan.SubscriptionPrice.Price); rerr != nil {
			c.log.Error("Failed to rollback balance", rerr)
		}

		c.processServers(sub, servers)
		return false
	}

	return true
}

func (c *Check) processServers(sub *models.Subscription, servers []*models.Server) {
	for _, serv := range servers {
		key, err := c.ks.Get(serv.ID, sub.UserID)
		if err != nil {
			c.log.Error("Failed to get server key", err, slog.Any("server", serv))
			continue
		}

		if err := c.api.DeleteRequest(serv, key.UUID); err != nil {
			c.log.Error("Failed to delete client", err, slog.Any("server", serv), slog.Any("key", key))
		} else {
			c.log.Info("Client deleted due to expiration", slog.Any("server", serv), slog.Any("key", key))
		}
	}

	if err := c.subs.UpdateIsActive(sub.ID, sub.UserID, false); err != nil {
		c.log.Error("Failed to deactivate subscription", err)
	}
}
