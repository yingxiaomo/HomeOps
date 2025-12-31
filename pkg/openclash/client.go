package openclash

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/yingxiaomo/homeops/config"
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

func (c *Client) request(method, endpoint string, body interface{}) (*http.Response, error) {
	var bodyReader *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBody)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, c.BaseURL+endpoint, bodyReader)
	if err != nil {
		return nil, err
	}

	if c.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.Secret)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.Client.Do(req)
}

func (c *Client) GetConfig() (map[string]interface{}, error) {
	resp, err := c.request("GET", "/configs", nil)
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

func (c *Client) PatchConfig(conf map[string]interface{}) error {
	resp, err := c.request("PATCH", "/configs", conf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetProxies() (map[string]interface{}, error) {
	resp, err := c.request("GET", "/proxies", nil)
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

func (c *Client) GetVersion() (map[string]interface{}, error) {
	resp, err := c.request("GET", "/version", nil)
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

func (c *Client) ReloadConfig() error {
	// PUT /configs?force=true
	// Body: {"path": "", "payload": ""}
	body := map[string]string{"path": "", "payload": ""}
	resp, err := c.request("PUT", "/configs?force=true", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) FlushFakeIP() error {
	resp, err := c.request("POST", "/cache/fakeip/flush", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) FlushConnections() error {
	resp, err := c.request("DELETE", "/connections", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetConnections() (map[string]interface{}, error) {
	resp, err := c.request("GET", "/connections", nil)
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

func (c *Client) PutProxy(group, node string) error {
	body := map[string]string{"name": node}
	// Escape the group name just in case
	endpoint := fmt.Sprintf("/proxies/%s", url.PathEscape(group))
	resp, err := c.request("PUT", endpoint, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetProxyDelay(node string) (int, error) {
	endpoint := fmt.Sprintf("/proxies/%s/delay?timeout=3000&url=http://www.gstatic.com/generate_204", url.PathEscape(node))
	resp, err := c.request("GET", endpoint, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}

	if delay, ok := res["delay"].(float64); ok {
		return int(delay), nil
	}
	return 0, fmt.Errorf("invalid response")
}
