package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/db"
	"nsvpn/pkg/logger"
	"time"
)

type Country struct{}

func NewCountry() *Country {
	return &Country{}
}

const (
	queryGetAllCountries   = `SELECT * FROM countries`
	queryGetCountryByCode  = `SELECT * FROM countries WHERE country_code = $1`
	queryAddCountry        = `INSERT INTO countries (country_code, country_name) VALUES ($1, $2) RETURNING id`
	queryUpdateCountryCode = `UPDATE countries SET country_code = $1 WHERE id = $2`
	queryUpdateCountryName = `UPDATE countries SET country_name = $1 WHERE id = $2`
	queryDeleteCountry     = `DELETE FROM countries WHERE id = $1`
)

func (c *Country) GetAll() (countries []models.Country, err error) {
	method := zap.String("method", "repository_Country_GetAll")

	cacheKey := "country:all"
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		logger.Error(constants.ErrGetDataFromCache, method, zap.Error(err))
		return nil, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &countries)
		if err != nil {
			logger.Error(constants.ErrUnmarshalDataFromJSON, method, zap.Error(err))
			return nil, err
		}
		return countries, nil
	}

	rows, err := db.Conn.Queryx(queryGetAllCountries)
	if err != nil {
		logger.Error(constants.ErrGetDataFromDB, method, zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var data models.Country
		if err := rows.StructScan(&data); err != nil {
			logger.Error(constants.ErrRowsScanFromDB, method, zap.Error(err))
			return nil, err
		}

		countries = append(countries, data)
	}

	jsonData, err := json.Marshal(countries)
	if err != nil {
		logger.Error(constants.ErrMarshalDataToJSON, method, zap.Error(err))
		return nil, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		logger.Error(constants.ErrSetDataToCache, method, zap.Error(err))
		return nil, err
	}

	return countries, nil
}

func (c *Country) Get(countryCode string) (country models.Country, err error) {
	method := zap.String("method", "repository_Country_Get")

	cacheKey := fmt.Sprintf("country:%s", countryCode)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		logger.Error(constants.ErrGetDataFromCache, method, zap.String("countryCode", countryCode), zap.Error(err))
		return models.Country{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &country)
		if err != nil {
			logger.Error(constants.ErrUnmarshalDataFromJSON, method, zap.String("countryCode", countryCode), zap.Error(err))
			return models.Country{}, err
		}
		return country, nil
	}

	err = db.Conn.QueryRowx(queryGetCountryByCode, countryCode).StructScan(&country)
	if err != nil {
		logger.Error(constants.ErrGetDataFromDB, method, zap.String("countryCode", countryCode), zap.Error(err))
		return models.Country{}, err
	}

	jsonData, err := json.Marshal(country)
	if err != nil {
		logger.Error(constants.ErrMarshalDataToJSON, method, zap.String("countryCode", countryCode), zap.Error(err))
		return models.Country{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		logger.Error(constants.ErrSetDataToCache, method, zap.String("countryCode", countryCode), zap.Error(err))
		return models.Country{}, err
	}

	return country, nil
}

func (c *Country) Add(country models.Country) (id int, err error) {
	method := zap.String("method", "repository_Country_Add")

	err = db.Conn.QueryRow(queryAddCountry, country.CountryCode, country.CountryName).Scan(&id)
	if err != nil {
		logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("country", country), zap.Error(err))
		return 0, err
	}

	return id, nil
}

func (c *Country) Update(country models.Country) error {
	method := zap.String("method", "repository_Country_Update")

	countryOld, err := c.Get(country.CountryCode)
	if err != nil {
		logger.Error("Error executing the Get function", method, zap.Any("country", country), zap.Error(err))
		return err
	}

	tx, err := db.Conn.Begin()
	if err != nil {
		logger.Error(constants.ErrBeginTx, method, zap.Any("country", country), zap.Any("countryOld", countryOld), zap.Error(err))
		return err
	}
	defer tx.Rollback()

	if country.CountryCode != countryOld.CountryCode && country.CountryCode != "" {
		_, err = tx.Exec(queryUpdateCountryCode, country.CountryCode, countryOld.ID)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("country", country), zap.Any("countryOld", countryOld), zap.Error(err))
			return err
		}
	}

	if country.CountryName != countryOld.CountryName && country.CountryName != "" {
		_, err = tx.Exec(queryUpdateCountryName, country.CountryCode, countryOld.ID)
		if err != nil {
			logger.Error(constants.ErrExecQueryFromDB, method, zap.Any("country", country), zap.Any("countryOld", countryOld), zap.Error(err))
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Error(constants.ErrCommitTx, method, zap.Any("country", country), zap.Any("countryOld", countryOld), zap.Error(err))
		return err
	}

	cacheKey := fmt.Sprintf("country:%s", country.CountryCode)
	err = cache.Rdb.Del(cache.Ctx, cacheKey).Err()
	if err != nil {
		logger.Error(constants.ErrDeleteDataFromCache, method, zap.Any("country", country), zap.Any("countryOld", countryOld), zap.Error(err))
		return err
	}

	return nil
}

func (c *Country) Delete(countryCode string) (err error) {
	method := zap.String("method", "repository_Country_Delete")

	_, err = db.Conn.Exec(queryDeleteCountry, countryCode)
	if err != nil {
		logger.Error(constants.ErrExecQueryFromDB, method, zap.String("countryCode", countryCode), zap.Error(err))
		return err
	}

	cacheKey := fmt.Sprintf("country:%s", countryCode)
	err = cache.Rdb.Del(cache.Ctx, cacheKey).Err()
	if err != nil {
		logger.Error(constants.ErrDeleteDataFromCache, method, zap.String("countryCode", countryCode), zap.Error(err))
		return err
	}

	return nil
}
