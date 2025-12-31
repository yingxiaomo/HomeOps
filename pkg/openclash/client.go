package openclash

import (
	"encoding/json"
	"fmt"
	"go_bot/config"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	Secret  string
	Client  *http.Client
}

func NewClient() *Client {
	return &Client{
		BaseURL: config.AppConfig.OpenClashAPIURL,
		Secret:  config.AppConfig.OpenClashAPISecret,
		Client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) GetConfig() (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/configs", nil)
	if err != nil {
		return nil, err
	}
	
	if c.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.Secret)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res, nil
}
