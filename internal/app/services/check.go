package services

import (
	"gopkg.in/telebot.v4"
	"log/slog"
	"nsvpn/internal/app/api"
	"nsvpn/pkg/logger"
	"time"
)

type Check struct {
	log           *logger.Logger
	b             *telebot.Bot
	ks            *Keys
	subs          *Subscriptions
	servs         *Servers
	api           *api.API
	clientButtons *Buttons
}

func NewCheck(log *logger.Logger, b *telebot.Bot, ks *Keys, subs *Subscriptions, servs *Servers, api *api.API, clientButtons *Buttons) *Check {
	return &Check{
		log:           log,
		b:             b,
		ks:            ks,
		subs:          subs,
		servs:         servs,
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
		expireTime := time.Until(sub.EndDate)

		isExpired := (expireTime <= 3*time.Hour && expireTime > 2*time.Hour) ||
			(expireTime <= 24*time.Hour && expireTime > 23*time.Hour) ||
			(expireTime <= 72*time.Hour && expireTime > 71*time.Hour) ||
			(expireTime <= 168*time.Hour && expireTime > 167*time.Hour) ||
			(sub.EndDate.Before(time.Now()) && !sub.EndDate.IsZero())

		if !isExpired {
			continue
		}

		for _, serv := range servers {
			key, err := c.ks.Get(serv.ID, sub.UserID)
			if err != nil {
				c.log.Error("Failed to get server key", err, slog.Any("server", serv))
				continue
			}

			c.log.Debug("Client has expired, attempting to delete")
			err = c.api.DeleteRequest(serv, key.UUID)
			if err != nil {
				c.log.Error("Failed to delete client", err, slog.Any("server", serv), slog.Any("key", key))
			} else {
				c.log.Info("Client deleted due to expiration", slog.Any("server", serv), slog.Any("key", key))
			}

			err = c.subs.UpdateIsActive(sub.ID, sub.UserID, false)
			if err != nil {
				c.log.Error("Failed to update subscription", err, slog.Any("server", serv), slog.Any("key", key))
			}
		}

		msg := "Ваша подписка истечёт в " + sub.EndDate.Format("2006-01-02 15:04:05")
		var opts *telebot.ReplyMarkup
		if sub.EndDate.Before(time.Now()) && !sub.EndDate.IsZero() {
			msg = "Ваша подписка истекла"
			opts = c.clientButtons.AddBtns()
		}

		_, err = c.b.Send(&telebot.User{ID: sub.UserID}, msg, opts)
		if err != nil {
			c.log.Error("Failed to send message", err)
		}
	}
}
