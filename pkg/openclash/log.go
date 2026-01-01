package openclash

import (
	"fmt"
	"time"

	"github.com/yingxiaomo/homeops/pkg/openwrt"
)

// GetDiagnosticLogs fetches a multi-source diagnostic log from the system.
// If setDebugLevel is true, it will temporarily switch OpenClash to debug log level
// to gather more detailed information, and then switch it back.
func GetDiagnosticLogs(setDebugLevel bool) (string, error) {
	client := NewClient()

	originalLevel := "info"
	// Only perform level switching if requested
	if setDebugLevel {
		config, err := client.GetConfig()
		if err == nil && config != nil {
			if l, ok := config["log-level"].(string); ok {
				originalLevel = l
			}
		}

		if originalLevel != "debug" {
			client.PatchConfig(map[string]interface{}{"log-level": "debug"})
			// Wait for a moment to allow new logs to be generated
			time.Sleep(5 * time.Second)
		}
	}

	// Ensure log level is restored if it was changed
	defer func() {
		if setDebugLevel && originalLevel != "debug" {
			client.PatchConfig(map[string]interface{}{"log-level": originalLevel})
		}
	}()

	diagCmd := "echo '--- [KERNEL LOG] ---'; tail -n 100 /tmp/openclash.log 2>/dev/null; " +
		"echo '--- [STARTUP/PLUGIN LOG] ---'; tail -n 100 /tmp/openclash_start.log 2>/dev/null; " +
		"echo '--- [SYSTEM SYSLOG] ---'; logread | grep -E -i 'clash|openclash' | tail -n 100; " +
		"echo '--- [NETWORK STATUS] ---'; ubus call network.interface.wan status | grep -E 'up|address|pending'"

	logs, err := openwrt.SSHExec(diagCmd)
	if err != nil {
		return "", err
	}
	if logs == "" {
		return "", fmt.Errorf("failed to collect any logs")
	}

	return logs, nil
}
