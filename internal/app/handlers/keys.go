package handlers

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"gorm.io/gorm"
	"nsvpn/internal/app/api"
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
	sub, err := k.ss.GetLastByUserID(c.Sender().ID, true)
	if err != nil {
		k.log.Error("Failed get last sub", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}
	if sub == nil || !(sub.EndDate.After(time.Now().UTC()) && sub.IsActive) {
		return c.Send("У вас нет прав для выполнения данной команды")
	}

	key, err := k.getOrCreateKey(c.Sender().ID, country.ID)
	if err != nil {
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	servers, err := k.servs.GetAllByCountryID(country.ID)
	if err != nil {
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	email := fmt.Sprintf("nsvpn-%d-%s", c.Sender().ID, strings.ToLower(country.Code))
	var wg sync.WaitGroup
	for _, serv := range servers {
		wg.Add(1)
		go func(server *models.Server) {
			defer wg.Done()

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
		}(serv)
	}
	wg.Wait()

	updateBtn := services.NewButtons([]models.ButtonOption{{
		Value:   "update_" + key.UUID,
		Display: "🔄 Обновить ключ",
	}}, []int{1}, "inline")
	for _, btn := range updateBtn.GetBtns() {
		k.bot.Handle(btn, func(c telebot.Context) error {
			return k.UpdateKeyHandler(c, country, servers, email, sub.EndDate)
		})
	}

	keyMessage := k.ss.GetVlessKey(key.UUID, country, email)
	return c.Send(fmt.Sprintf("Ваш ключ для сервера %s %s:\n```%s```", country.Emoji, country.Code, keyMessage), &telebot.SendOptions{
		ReplyMarkup: updateBtn.AddBtns(),
		ParseMode:   telebot.ModeMarkdown,
	})
}

func (k *Keys) UpdateKeyHandler(c telebot.Context, country *models.Country, servers []*models.Server, email string, endDate time.Time) error {
	u := strings.TrimPrefix(c.Callback().Unique, "update_")
	newUUID, _ := uuid.NewUUID()
	err := k.ks.Update(country.ID, c.Sender().ID, &models.Key{UUID: newUUID.String()})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, serv := range servers {
		wg.Add(1)
		go func(server *models.Server) {
			defer wg.Done()

			if err := k.api.DeleteRequest(server, u); err != nil && err.Error() != "record not found" {
				k.log.Error("Failed delete request", err)
				return
			}

			if err := k.api.AddRequest(server, newUUID.String(), email, endDate); err != nil {
				k.log.Error("Failed add request", err)
				return
			}
		}(serv)
	}

	if err = k.ks.Delete(country.ID, c.Sender().ID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		k.log.Error("Failed delete request", err)
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	if c.Message() != nil {
		err = k.bot.Delete(c.Message())
		if err != nil {
			k.log.Error("Failed to delete message", err)
		}
	}

	keyMessage := k.ss.GetVlessKey(newUUID.String(), country, email)
	return c.Send(fmt.Sprintf("Ваш новый ключ для сервера %s %s:\n```%s```", country.Emoji, country.Code, keyMessage), telebot.ModeMarkdown)
}

func (k *Keys) getOrCreateKey(userID int64, countryID int) (*models.Key, error) {
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
