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
