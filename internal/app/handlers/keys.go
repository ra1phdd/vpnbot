package handlers

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
	"nsvpn/internal/app/api"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"strings"
	"sync"
	"time"
)

type Keys struct {
	log   *logger.Logger
	bot   *telebot.Bot
	ks    *services.Keys
	servs *services.Servers
	ss    *services.Subscriptions
	api   *api.API
}

func NewKeys(log *logger.Logger, bot *telebot.Bot, ks *services.Keys, servs *services.Servers, ss *services.Subscriptions, api *api.API) *Keys {
	return &Keys{
		log:   log,
		bot:   bot,
		ks:    ks,
		servs: servs,
		ss:    ss,
		api:   api,
	}
}

func (k *Keys) GetKeyHandler(c telebot.Context, country *models.Country) error {
	btns := getReplyButtons(c)
	sub, subOk := c.Get("sub").(*models.Subscription)
	if subOk && !sub.IsActive && (sub.EndDate.Before(time.Now()) && (!sub.EndDate.IsZero() || sub.ID == 0)) {
		return c.Send(constants.UserError, btns)
	}

	key, err := k.getOrCreateKey(c.Sender().ID, country.ID)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	servers, err := k.servs.GetAllByCountryID(country.ID)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	email := fmt.Sprintf("nsvpn-%d-%s", c.Sender().ID, strings.ToLower(country.Code))
	k.processServers(servers, func(server *models.Server) {
		found, err := k.api.IsFoundRequest(server, key.UUID)
		if err != nil {
			k.log.Error("Failed check if request", err)
			return
		}

		if found {
			return
		}

		if err := k.api.AddRequest(server, key.UUID, email, sub.EndDate); err != nil {
			k.log.Error("Failed add request", err)
			return
		}
	})

	updateBtn := services.NewButtons([]models.ButtonOption{{
		Value:   "update_" + key.UUID,
		Display: "🔄 Обновить ключ",
	}}, []int{1}, "inline")
	for _, btn := range updateBtn.GetBtns() {
		k.bot.Handle(btn, func(c telebot.Context) error {
			return k.UpdateKeyHandler(c, country, servers, email, sub.EndDate)
		})
	}

	keyMessage := k.ks.GetVlessKey(key.UUID, country, email)
	return c.Send(fmt.Sprintf("🔑 Ваш ключ для сервера %s %s:\n```%s```", country.Emoji, country.Code, keyMessage), &telebot.SendOptions{
		ReplyMarkup: updateBtn.AddBtns(),
		ParseMode:   telebot.ModeMarkdown,
	})
}

func (k *Keys) UpdateKeyHandler(c telebot.Context, country *models.Country, servers []*models.Server, email string, endDate time.Time) error {
	btns := getReplyButtons(c)
	if err := validateSubscription(c); err != nil {
		return err
	}

	u := strings.TrimPrefix(c.Callback().Unique, "update_")
	newUUID := uuid.New().String()

	err := k.ks.Update(country.ID, c.Sender().ID, &models.Key{UUID: newUUID})
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	k.processServers(servers, func(server *models.Server) {
		if err := k.api.DeleteRequest(server, u); err != nil && err.Error() != "record not found" {
			k.log.Error("Failed delete request", err)
			return
		}

		if err := k.api.AddRequest(server, newUUID, email, endDate); err != nil {
			k.log.Error("Failed add request", err)
		}
	})

	if err = k.ks.Delete(country.ID, c.Sender().ID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		k.log.Error("Failed delete request", err)
		return c.Send(constants.UserError, btns)
	}

	if c.Message() != nil {
		err = k.bot.Delete(c.Message())
		if err != nil {
			k.log.Error("Failed to delete message", err)
		}
	}

	keyMessage := k.ks.GetVlessKey(newUUID, country, email)
	return c.Send(fmt.Sprintf("🔑 Ваш новый ключ для сервера %s %s:\n```%s```", country.Emoji, country.Code, keyMessage), telebot.ModeMarkdown)
}

func (k *Keys) getOrCreateKey(userID int64, countryID uint) (*models.Key, error) {
	key, err := k.ks.Get(countryID, userID)
	if err != nil {
		k.log.Error("Failed get key", err)
		return nil, err
	}
	if key != nil {
		return key, nil
	}

	newKey := &models.Key{
		UserID:       userID,
		CountryID:    countryID,
		UUID:         uuid.New().String(),
		SpeedLimit:   0,
		TrafficLimit: 0,
		TrafficUsed:  0,
		IsActive:     true,
	}

	if err := k.ks.Add(newKey); err != nil {
		k.log.Error("Failed add key", err)
		return nil, err
	}

	return newKey, nil
}

func (k *Keys) processServers(servers []*models.Server, process func(server *models.Server)) {
	var wg sync.WaitGroup
	for _, server := range servers {
		wg.Add(1)
		go func(server *models.Server) {
			defer wg.Done()
			process(server)
		}(server)
	}
	wg.Wait()
}
