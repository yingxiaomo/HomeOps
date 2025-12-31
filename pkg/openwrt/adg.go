package openwrt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yingxiaomo/homeops/config"
)

type AdGuardClient struct {
	BaseURL  string
	Username string
	Password string
	Token    string
	Client   *http.Client
}

func NewAdGuardClient() *AdGuardClient {
	return &AdGuardClient{
		BaseURL:  config.AppConfig.AdgURL,
		Username: config.AppConfig.AdgUser,
		Password: config.AppConfig.AdgPass,
		Token:    config.AppConfig.AdgToken,
		Client:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *AdGuardClient) Request(method, endpoint string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	} else if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (c *AdGuardClient) GetDHCPLeases() ([]map[string]interface{}, error) {
	res, err := c.Request("GET", "/control/dhcp/status", nil)
	if err == nil {
		var status struct {
			Leases []map[string]interface{} `json:"leases"`
		}
		if err := json.Unmarshal(res, &status); err == nil && len(status.Leases) > 0 {
			return status.Leases, nil
		}
	}

	if config.AppConfig.AdgLeasesMode == "api" {
		return nil, fmt.Errorf("API returned no leases and SSH fallback disabled")
	}

	paths := []string{
		"/var/lib/AdGuardHome/dhcp.leases",
		"/var/lib/adguardhome/dhcp.leases",
		"/tmp/AdGuardHome/dhcp.leases",
	}

	for _, p := range paths {
		content, _ := SSHExec(fmt.Sprintf("cat %s 2>/dev/null", p))
		if content != "" {
			leases := []map[string]interface{}{}
			lines := strings.Split(content, "\n")
			for _, ln := range lines {
				parts := strings.Fields(ln)
				if len(parts) >= 4 {
					leases = append(leases, map[string]interface{}{
						"mac":      parts[1],
						"ip":       parts[2],
						"hostname": parts[3],
					})
				} else if len(parts) >= 3 {
					leases = append(leases, map[string]interface{}{
						"ip":       parts[0],
						"mac":      parts[1],
						"hostname": parts[2],
					})
				}
			}
			if len(leases) > 0 {
				return leases, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to get leases from API and SSH")
}

func (c *AdGuardClient) GetFilteringStatus() (bool, error) {
	res, err := c.Request("GET", "/control/filtering/status", nil)
	if err != nil {
		return false, err
	}

	var status struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(res, &status); err != nil {
		return false, err
	}
	return status.Enabled, nil
}

func (c *AdGuardClient) SetFiltering(enabled bool) error {
	body := map[string]bool{"enabled": enabled}
	_, err := c.Request("POST", "/control/filtering/config", body)
	return err
}

func (c *AdGuardClient) GetStats() (map[string]interface{}, error) {
	res, err := c.Request("GET", "/control/stats", nil)
	if err != nil {
		return nil, err
	}
	var stats map[string]interface{}
	if err := json.Unmarshal(res, &stats); err != nil {
		return nil, err
	}
	return stats, nil
}

func (c *AdGuardClient) GetFeatureStatus(endpoint string) (bool, error) {
	res, err := c.Request("GET", endpoint, nil)
	if err != nil {
		return false, err
	}
	var status struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(res, &status); err != nil {
		return false, err
	}
	return status.Enabled, nil
}

func (c *AdGuardClient) SetFeatureStatus(endpoint string, enabled bool) error {
	action := "disable"
	if enabled {
		action = "enable"
	}
	_, err := c.Request("POST", fmt.Sprintf("%s/%s", endpoint, action), map[string]interface{}{})
	return err
}

func (c *AdGuardClient) GetConfig(endpoint string) (map[string]interface{}, error) {
	res, err := c.Request("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(res, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *AdGuardClient) SetConfig(endpoint string, cfg map[string]interface{}) error {
	_, err := c.Request("POST", endpoint, cfg)
	return err
}

func (c *AdGuardClient) GetDNSInfo() (map[string]interface{}, error) {
	res, err := c.Request("GET", "/control/dns_info", nil)
	if err != nil {
		return nil, err
	}
	var info map[string]interface{}
	if err := json.Unmarshal(res, &info); err != nil {
		return nil, err
	}
	return info, nil
}

func (c *AdGuardClient) SetDNSConfig(cfg map[string]interface{}) error {
	return c.SetConfig("/control/dns_config", cfg)
}

func (c *AdGuardClient) GetDHCPStatus() (map[string]interface{}, error) {
	res, err := c.Request("GET", "/control/dhcp/status", nil)
	if err != nil {
		return nil, err
	}
	var status map[string]interface{}
	if err := json.Unmarshal(res, &status); err != nil {
		return nil, err
	}
	return status, nil
}

func (c *AdGuardClient) SetDHCPConfig(cfg map[string]interface{}) error {
	_, err := c.Request("POST", "/control/dhcp/set_config", cfg)
	return err
}

func (c *AdGuardClient) GetFiltering() (map[string]interface{}, error) {
	res, err := c.Request("GET", "/control/filtering/status", nil)
	if err != nil {
		return nil, err
	}
	var status map[string]interface{}
	if err := json.Unmarshal(res, &status); err != nil {
		return nil, err
	}
	return status, nil
}

func (c *AdGuardClient) AddFilter(name, url string, whitelist bool) error {
	body := map[string]interface{}{
		"name":      name,
		"url":       url,
		"whitelist": whitelist,
	}
	_, err := c.Request("POST", "/control/filtering/add_url", body)
	return err
}

func (c *AdGuardClient) RemoveFilter(url string, whitelist bool) error {
	body := map[string]interface{}{
		"url":       url,
		"whitelist": whitelist,
	}
	_, err := c.Request("POST", "/control/filtering/remove_url", body)
	return err
}

func (c *AdGuardClient) SetRules(rules []string) error {
	body := map[string]interface{}{
		"rules": rules,
	}
	_, err := c.Request("POST", "/control/filtering/set_rules", body)
	return err
}
