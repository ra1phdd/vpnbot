package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/db"
)

type Users struct{}

func NewUsers() *Users {
	return &Users{}
}

func (u *Users) GetById(id int64) (models.User, error) {
	var data models.User

	cacheKey := fmt.Sprintf("user:%d", id)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return models.User{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &data)
		if err != nil {
			return models.User{}, err
		}
	}

	err = db.Conn.QueryRowx(`SELECT * FROM users WHERE id = $1`, id).StructScan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, constants.ErrUserNotFound
		}
		return models.User{}, err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return models.User{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 0).Err()
	if err != nil {
		return models.User{}, err
	}

	return data, nil
}

func (u *Users) Update(user models.User) error {
	userOld, err := u.GetById(user.ID)
	if err != nil {
		return err
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if user.Username != userOld.Username {
		_, err = tx.Exec(`UPDATE users SET username = $1 WHERE id = $2`, user.Username, user.ID)
		if err != nil {
			return err
		}
	}

	if user.Firstname != userOld.Firstname {
		_, err = tx.Exec(`UPDATE users SET firstname = $1 WHERE id = $2`, user.Firstname, user.ID)
		if err != nil {
			return err
		}
	}

	if user.Lastname != userOld.Lastname {
		_, err = tx.Exec(`UPDATE users SET lastname = $1 WHERE id = $2`, user.Lastname, user.ID)
		if err != nil {
			return err
		}
	}

	if user.IsAdmin != userOld.IsAdmin {
		_, err = tx.Exec(`UPDATE users SET is_admin = $1 WHERE id = $2`, user.IsAdmin, user.ID)
		if err != nil {
			return err
		}
	}

	if user.IsSign != userOld.IsSign {
		_, err = tx.Exec(`UPDATE users SET is_sign = $1 WHERE id = $2`, user.IsSign, user.ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (u *Users) Add(user models.User) error {
	var partnerID interface{}
	if user.PartnerID != nil {
		partnerID = *user.PartnerID
	} else {
		partnerID = nil
	}

	_, err := db.Conn.Exec(`INSERT INTO users (id, username, firstname, lastname, partner_id) VALUES ($1, $2, $3, $4, $5)`, user.ID, user.Username, user.Firstname, user.Lastname, partnerID)
	return err
}
