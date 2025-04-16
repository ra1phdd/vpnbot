package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"nsvpn/internal/app/models"
	"time"
)

type ServerAPI struct {
	server  models.Server
	authKey [32]byte
}

func NewServer(server models.Server) *ServerAPI {
	sa := &ServerAPI{
		server: server,
	}
	sa.authKey = sha256.Sum256([]byte(fmt.Sprintf("%s%s", server.PublicKey, server.PrivateKey)))

	return sa
}

func (sa *ServerAPI) IsFoundRequest(uuid, email string) (bool, error) {
	endpoint := fmt.Sprintf("http://%s:%d/v1/client/is_found", sa.server.IP, sa.server.Port)
	params := url.Values{}
	params.Add("uuid", uuid)
	params.Add("email", email)

	result, err := sa.serverRequest(endpoint, params)
	if err != nil {
		return false, err
	}

	if found, ok := result["found"]; ok {
		return found.(bool), nil
	}
	return false, nil
}

func (sa *ServerAPI) AddRequest(uuid, email string, expiresAt time.Time) error {
	endpoint := fmt.Sprintf("http://%s:%d/v1/client/add", sa.server.IP, sa.server.Port)
	params := url.Values{}
	params.Add("uuid", uuid)
	params.Add("email", email)
	params.Add("expires_at", expiresAt.Format(time.RFC3339))

	result, err := sa.serverRequest(endpoint, params)
	if err != nil {
		return err
	}

	if errResult := result["error"].(string); errResult != "" {
		return errors.New(errResult)
	}
	return nil
}

func (sa *ServerAPI) GetLoadRequest() (float64, error) {
	endpoint := fmt.Sprintf("http://%s:%d/v1/server/get_load", sa.server.IP, sa.server.Port)
	result, err := sa.serverRequest(endpoint, url.Values{})
	if err != nil {
		return 0, err
	}

	if score, ok := result["totalScore"]; ok {
		return score.(float64), nil
	}
	return 0, nil
}

func (sa *ServerAPI) serverRequest(endpoint string, params url.Values) (map[string]interface{}, error) {
	uri := endpoint
	if params.Encode() != "" {
		uri = endpoint + "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-AUTH-KEY", hex.EncodeToString(sa.authKey[:]))

	client := &http.Client{
		Timeout: 1 * time.Second,
	}
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
