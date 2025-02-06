package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"strings"
)

type Keys struct {
	ks   *services.Keys
	ss   *services.Servers
	subs *services.Subscriptions
}

func NewKeys(ks *services.Keys, ss *services.Servers, subs *services.Subscriptions) *Keys {
	return &Keys{
		ks:   ks,
		ss:   ss,
		subs: subs,
	}
}

func (k *Keys) GetKeyHandler(c telebot.Context, server models.Server, countryName string, btnUnique string) error {
	email := fmt.Sprintf("%d-%s", c.Sender().ID, strings.ToLower(btnUnique))

	key, err := k.ks.GetByServerId(server.ID, c.Sender().ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	var uuidStr string
	if errors.Is(err, sql.ErrNoRows) {
		u, err := uuid.NewUUID()
		if err != nil {
			return err
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
			return err
		}
	} else {
		uuidStr = key.UUID
	}

	authKey := sha256.Sum256([]byte(fmt.Sprintf("%s%s", server.PublicKey, server.PrivateKey)))

	found, err := k.ss.IsFoundRequest(server.IP, server.Port, hex.EncodeToString(authKey[:]), uuidStr, email)
	if err != nil {
		return err
	}

	if !found {
		sub, err := k.subs.GetLastByUserId(c.Sender().ID)
		if err != nil {
			return err
		}

		if err := k.ss.AddRequest(server.IP, server.Port, hex.EncodeToString(authKey[:]), uuidStr, email, *sub.EndDate); err != nil {
			return err
		}
	}

	keyMessage := k.subs.GetVlessKey(uuidStr, server, email)
	return c.Send(fmt.Sprintf("Ваш ключ для сервера %s:\n```%s```", countryName, keyMessage), telebot.ModeMarkdown)
}
