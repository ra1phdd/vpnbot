package handlers

import (
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/api"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"strings"
)

type Keys struct {
	log *logger.Logger
	ks  *services.Keys
	ss  *services.Subscriptions
}

func NewKeys(log *logger.Logger, ks *services.Keys, ss *services.Subscriptions) *Keys {
	return &Keys{
		log: log,
		ks:  ks,
		ss:  ss,
	}
}

func (k *Keys) GetKeyHandler(c telebot.Context, server models.Server, countryName string, btnUnique string) error {
	email := fmt.Sprintf("nsvpn-%d-%s", c.Sender().ID, strings.ToLower(btnUnique))

	key, err := k.ks.Get(server.ID, c.Sender().ID)
	if err != nil {
		k.log.Error("Failed get key", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	var uuidStr string
	if key == (models.Key{}) {
		u, err := uuid.NewUUID()
		if err != nil {
			k.log.Error("Failed get uuid", err)
			return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
		}

		uuidStr = u.String()
		data := models.Key{
			UserID:       c.Sender().ID,
			ServerID:     server.ID,
			UUID:         uuidStr,
			SpeedLimit:   0,
			TrafficLimit: 0,
			TrafficUsed:  0,
			IsActive:     true,
		}

		if err := k.ks.Add(data); err != nil {
			k.log.Error("Failed add key", err)
			return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
		}
	} else {
		uuidStr = key.UUID
	}

	sa := api.NewAPI(k.log, server)
	found, err := sa.IsFoundRequest(uuidStr)
	if err != nil {
		k.log.Error("Failed check if request", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	if !found {
		sub, err := k.ss.GetLastByUserId(c.Sender().ID, true)
		if err != nil {
			k.log.Error("Failed get last sub", err)
			return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
		}

		if err := sa.AddRequest(uuidStr, email, *sub.EndDate); err != nil {
			k.log.Error("Failed add request", err)
			return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
		}
	}

	keyMessage := k.ss.GetVlessKey(uuidStr, server, email)
	return c.Send(fmt.Sprintf("Ваш ключ для сервера %s:\n```%s```", countryName, keyMessage), telebot.ModeMarkdown)
}
