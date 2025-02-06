package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/logger"
	"strconv"
	"time"
)

type Servers struct {
	sr *repository.Servers
	cr *repository.Country
}

func NewServers(sr *repository.Servers, cr *repository.Country) *Servers {
	return &Servers{
		sr: sr,
		cr: cr,
	}
}

func (s *Servers) GetByCC(countryCode string) ([]models.Server, error) {
	if countryCode == "" {
		return nil, fmt.Errorf("countryCode is empty")
	}

	country, err := s.cr.Get(countryCode)
	if err != nil {
		logger.Error("Couldn't get country information by code", zap.String("code", countryCode), zap.Error(err))
		return nil, err
	}

	return s.sr.GetByCC(country.ID)
}

func (s *Servers) ProcessServers(country models.Country, servers []models.Server) ([]models.ButtonOption, []int) {
	var listServers []models.ButtonOption

	for i, _ := range servers {
		listServers = append(listServers, models.ButtonOption{
			Value:   fmt.Sprintf("%s-%d", country.CountryCode, i+1),
			Display: fmt.Sprintf("%s-%d", country.CountryName, i+1),
		})
	}

	var groups []int
	remaining := len(listServers)
	for remaining > 0 {
		if remaining >= 4 {
			groups = append(groups, 4)
			remaining -= 4
		} else {
			groups = append(groups, remaining)
			break
		}
	}

	return listServers, groups
}

func (s *Servers) serverRequest(endpoint string, params url.Values, key string) (map[string]interface{}, error) {
	var uri string
	if params.Encode() == "" {
		uri = endpoint
	} else {
		uri = endpoint + "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-AUTH-KEY", key)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return result, nil
}

func (s *Servers) IsFoundRequest(ip string, port int, key, uuid, email string) (bool, error) {
	endpoint := fmt.Sprintf("http://%s:%d/v1/client/is_found", ip, port)
	params := url.Values{}
	params.Add("uuid", uuid)
	params.Add("email", email)

	result, err := s.serverRequest(endpoint, params, key)
	if err != nil {
		return false, err
	}

	if found, ok := result["found"]; ok {
		return found.(bool), nil
	}
	return false, nil
}

func (s *Servers) AddRequest(ip string, port int, key, uuid, email string, expiresAt time.Time) error {
	endpoint := fmt.Sprintf("http://%s:%d/v1/client/add", ip, port)
	params := url.Values{}
	params.Add("uuid", uuid)
	params.Add("email", email)
	params.Add("expires_at", expiresAt.Format(time.RFC3339))

	result, err := s.serverRequest(endpoint, params, key)
	if err != nil {
		return err
	}

	if errResult := result["error"].(string); errResult != "" {
		return fmt.Errorf(errResult)
	}
	return nil
}

func (s *Servers) GetLoadRequest(ip string, port int, key string) (float64, error) {
	cacheKey := fmt.Sprintf("servers_load:ip:%s", ip)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, err
	} else if cacheValue != "" {
		return strconv.ParseFloat(cacheValue, 64)
	}

	endpoint := fmt.Sprintf("http://%s:%d/v1/server/get_load", ip, port)
	result, err := s.serverRequest(endpoint, url.Values{}, key)
	if err != nil {
		return 0, err
	}

	if score, ok := result["totalScore"]; ok {
		err = cache.Rdb.Set(cache.Ctx, cacheKey, score, 5*time.Minute).Err()
		if err != nil {
			return 0, err
		}

		return score.(float64), nil
	}
	return 0, nil
}
