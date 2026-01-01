package openwrt

import (
	"fmt"
	"strings"
)

// GetLogs fetches the last N lines from OpenWrt's system log.
func GetLogs(count int) (string, error) {
	cmd := fmt.Sprintf("logread | tail -n %d", count)
	logs, err := SSHExec(cmd)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(logs) == "" {
		return "", fmt.Errorf("log is empty")
	}
	return logs, nil
}
