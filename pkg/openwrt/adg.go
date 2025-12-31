package openwrt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/yingxiaomo/HomeOps/config"
	"net/http"
	"strings"
	"time"
)

type AdGuardClient struct {
	BaseURL string
	User    string
	Pass    string
	Client  *http.Client
}

func NewAdGuardClient() *AdGuardClient {
	// Fallback logic for ADG URL
	url := config.AppConfig.ADG_URL
	if url == "" {
		url = fmt.Sprintf("http://%s:3000", config.AppConfig.OpenWrtHost)
	}

	return &AdGuardClient{
		BaseURL: strings.TrimRight(url, "/"),
		User:    config.AppConfig.ADG_USER,
		Pass:    config.AppConfig.ADG_PASS,
		Client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *AdGuardClient) Request(method, endpoint string) (map[string]interface{}, error) {
	req, err := http.NewRequest(method, c.BaseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Auth
	auth := base64.StdEncoding.EncodeToString([]byte(c.User + ":" + c.Pass))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var res map[string]interface{}
	// Some ADG endpoints return empty body or plain text
	if resp.ContentLength == 0 {
		return nil, nil
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		// Ignore json errors for now as some endpoints return arrays or text
		return nil, nil
	}
	return res, nil
}

func (c *AdGuardClient) GetStats() (map[string]interface{}, error) {
	return c.Request("GET", "/control/stats")
}

func (c *AdGuardClient) GetFilteringStatus() (bool, error) {
	res, err := c.Request("GET", "/control/filtering/status")
	if err != nil {
		return false, err
	}
	if enabled, ok := res["enabled"].(bool); ok {
		return enabled, nil
	}
	return false, nil
}

func (c *AdGuardClient) SetFiltering(enabled bool) error {
	url := "/control/filtering/enable"
	if !enabled {
		url = "/control/filtering/disable"
	}
	req, err := http.NewRequest("POST", c.BaseURL+url, nil)
	if err != nil {
		return err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(c.User + ":" + c.Pass))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}
