package handlers

import (
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/api"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/internal/app/state"
	"nsvpn/pkg/logger"
	"strconv"
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

	GetBtns    *services.Buttons
	updateBtns *services.Buttons

	KeysState state.Storage[state.KeysState]
}

func NewKeys(log *logger.Logger, bot *telebot.Bot, ks *services.Keys, servs *services.Servers, ss *services.Subscriptions, api *api.API) *Keys {
	return &Keys{
		log:   log,
		bot:   bot,
		ks:    ks,
		servs: servs,
		ss:    ss,
		api:   api,
		GetBtns: services.NewButtons([]models.ButtonOption{{
			Value:   "get_key",
			Display: "📥 Получить ключ",
		}}, []int{1}, "inline"),
		updateBtns: services.NewButtons([]models.ButtonOption{{
			Value:   "update_key",
			Display: "🔄 Обновить ключ",
		}}, []int{1}, "inline"),
		KeysState: state.NewMemoryStorage[state.KeysState](),
	}
}

func (k *Keys) RegisterHandlers() {
	k.bot.Handle(k.GetBtns.GetBtn("get_key"), k.GetKeyHandler)
	k.bot.Handle(k.updateBtns.GetBtn("update_key"), k.UpdateKeyHandler)
}

func (k *Keys) GetKeyHandler(c telebot.Context) error {
	defer func(c telebot.Context) {
		if err := c.Respond(); err != nil {
			k.log.Error("Failed to send message", err)
		}
	}(c)

	btns := getReplyButtons(c)
	sub := getSubscription(c, k.ss)
	if sub == nil || !sub.IsActive || (sub.EndDate.Before(time.Now()) && (!sub.EndDate.IsZero() || sub.ID == 0)) {
		return c.Send(constants.UserError, btns)
	}

	ks, exists := k.KeysState.Get(strconv.FormatInt(c.Sender().ID, 10))
	if !exists {
		return c.Send(constants.UserError, btns)
	}

	key, err := k.getOrCreateKey(c.Sender().ID, ks.Country.ID)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	servers, err := k.servs.GetAllByCountryID(ks.Country.ID)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	email := fmt.Sprintf("nsvpn-%d-%s", c.Sender().ID, strings.ToLower(ks.Country.Code))
	err = k.processServers(servers, func(server *models.Server) error {
		found, err := k.api.IsFoundRequest(server, key.UUID)
		if err != nil {
			k.log.Error("Failed check if request", err)
			return err
		}
		if found {
			return nil
		}

		if err := k.api.AddRequest(server, key.UUID, email, sub.EndDate); err != nil {
			k.log.Error("Failed add request", err)
			return err
		}

		return nil
	})
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	k.KeysState.Update(strconv.FormatInt(c.Sender().ID, 10), func(ks state.KeysState) state.KeysState {
		ks.UUID = key.UUID
		ks.Email = email
		ks.EndDate = sub.EndDate
		ks.Servers = servers
		return ks
	})

	keyMessage := k.ks.GetVlessKey(key.UUID, ks.Country, email)
	return c.Send(fmt.Sprintf("🔑 Ваш ключ для сервера %s %s:\n```%s```", ks.Country.Emoji, ks.Country.Code, keyMessage), &telebot.SendOptions{
		ReplyMarkup: k.updateBtns.AddBtns(),
		ParseMode:   telebot.ModeMarkdown,
	})
}

func (k *Keys) UpdateKeyHandler(c telebot.Context) error {
	defer func(c telebot.Context) {
		if err := c.Respond(); err != nil {
			k.log.Error("Failed to send message", err)
		}
	}(c)

	btns := getReplyButtons(c)
	if err := validateSubscription(c, k.ss); err != nil {
		return err
	}

	ks, exists := k.KeysState.Get(strconv.FormatInt(c.Sender().ID, 10))
	if !exists {
		return c.Send(constants.UserError, btns)
	}
	newUUID := uuid.New().String()

	err := k.ks.Update(ks.Country.ID, c.Sender().ID, &models.Key{UUID: newUUID})
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	err = k.processServers(ks.Servers, func(server *models.Server) error {
		if err := k.api.DeleteRequest(server, ks.UUID); err != nil && err.Error() != "record not found" {
			k.log.Error("Failed delete request", err)
			return err
		}

		if err := k.api.AddRequest(server, newUUID, ks.Email, ks.EndDate); err != nil {
			k.log.Error("Failed add request", err)
			return err
		}

		return nil
	})
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	if c.Message() != nil {
		err = k.bot.Delete(c.Message())
		if err != nil {
			k.log.Error("Failed to delete message", err)
		}
	}

	k.KeysState.Delete(strconv.FormatInt(c.Sender().ID, 10))
	keyMessage := k.ks.GetVlessKey(newUUID, ks.Country, ks.Email)
	return c.Send(fmt.Sprintf("🔑 Ваш новый ключ для сервера %s %s:\n```%s```", ks.Country.Emoji, ks.Country.Code, keyMessage), telebot.ModeMarkdown)
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

func (k *Keys) processServers(servers []*models.Server, process func(server *models.Server) error) error {
	var wg sync.WaitGroup
	var errs []error
	for _, server := range servers {
		wg.Add(1)
		go func(server *models.Server) {
			defer wg.Done()
			if err := process(server); err != nil {
				errs = append(errs, err)
			}
		}(server)
	}
	wg.Wait()

	if len(errs) > 0 {
		for _, err := range errs {
			k.log.Error("Failed process servers", err)
		}
		return constants.ErrProcessServers
	}
	return nil
}
