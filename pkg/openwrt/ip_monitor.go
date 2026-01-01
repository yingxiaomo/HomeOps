package openwrt

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/yingxiaomo/homeops/config"
	tele "gopkg.in/telebot.v3"
)

const IPHistoryFile = "data/ip_history.json"

type IPHistory struct {
	V4 string `json:"v4"`
	V6 string `json:"v6"`
}

func GetRouterIPs() (string, string) {
	v4 := ""
	v6 := ""
	ifaces := []string{"wan", "wan_6", "wan6"}

	for _, iface := range ifaces {
		cmd := fmt.Sprintf("ubus call network.interface.%s status", iface)
		res, _ := SSHExec(cmd)
		if res == "" {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(res), &data); err == nil {
			if v4 == "" {
				if ipv4Arr, ok := data["ipv4-address"].([]interface{}); ok && len(ipv4Arr) > 0 {
					if addrMap, ok := ipv4Arr[0].(map[string]interface{}); ok {
						if addr, ok := addrMap["address"].(string); ok && strings.Contains(addr, ".") {
							v4 = addr
						}
					}
				}
			}
			if v6 == "" {
				if ipv6Arr, ok := data["ipv6-address"].([]interface{}); ok {
					for _, item := range ipv6Arr {
						if addrMap, ok := item.(map[string]interface{}); ok {
							if addr, ok := addrMap["address"].(string); ok {
								if strings.Contains(addr, ":") && !strings.HasPrefix(addr, "fe80") {
									v6 = addr
									break
								}
							}
						}
					}
				}
				if v6 == "" {
					if ipv6Pre, ok := data["ipv6-prefix-assignment"].([]interface{}); ok {
						for _, item := range ipv6Pre {
							if prefixMap, ok := item.(map[string]interface{}); ok {
								if addr, ok := prefixMap["address"].(string); ok && !strings.HasPrefix(addr, "fe80") {
									v6 = addr
									break
								}
							}
						}
					}
				}
			}
		}
	}

	if v4 == "" {
		res, _ := SSHExec("/usr/bin/curl -4 -s --max-time 5 icanhazip.com || /usr/bin/curl -4 -s --max-time 5 ifconfig.me")
		if res != "" && strings.Contains(res, ".") {
			v4 = strings.TrimSpace(res)
		}
	}
	if v6 == "" {
		res, _ := SSHExec("/usr/bin/curl -6 -s --max-time 5 icanhazip.com || /usr/bin/curl -6 -s --max-time 5 ifconfig.co")
		if res != "" && strings.Contains(res, ":") {
			v6 = strings.TrimSpace(res)
		}
	}

	if len(v4) > 15 || !strings.Contains(v4, ".") {
		v4 = ""
	}
	if !strings.Contains(v6, ":") {
		v6 = ""
	}

	return v4, v6
}

func getStoredIPs() *IPHistory {
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		os.Mkdir("data", 0755)
	}
	data, err := os.ReadFile(IPHistoryFile)
	if err != nil {
		return &IPHistory{}
	}
	var history IPHistory
	json.Unmarshal(data, &history)
	return &history
}

func saveStoredIPs(history *IPHistory) {
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		os.Mkdir("data", 0755)
	}
	data, _ := json.Marshal(history)
	os.WriteFile(IPHistoryFile, data, 0644)
}

func StartIPMonitor(b *tele.Bot) {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for range ticker.C {
			checkIPJob(b)
		}
	}()
	log.Println("IP Monitor Job registered.")
}

func checkIPJob(b *tele.Bot) {
	currentV4, currentV6 := GetRouterIPs()
	if currentV4 == "" && currentV6 == "" {
		return
	}

	stored := getStoredIPs()
	changed := false
	msg := "🚨 **公网 IP 变动通知**\n-------------------\n"

	if currentV4 != "" && currentV4 != stored.V4 {
		old := stored.V4
		if old == "" {
			old = "未知"
		}
		msg += fmt.Sprintf("🔴 IPv4: `%s`\n(旧: %s)\n", currentV4, old)
		stored.V4 = currentV4
		changed = true
	}

	if currentV6 != "" && currentV6 != stored.V6 {
		old := stored.V6
		if old == "" {
			old = "未知"
		}
		msg += fmt.Sprintf("🔵 IPv6: `%s`\n(旧: %s)\n", currentV6, old)
		stored.V6 = currentV6
		changed = true
	}

	if changed {
		saveStoredIPs(stored)
		adminID := config.AppConfig.AdminID
		if adminID != 0 {
			menu := &tele.ReplyMarkup{}
			menu.Inline(menu.Row(menu.Data("🔙 返回主菜单", "start_main")))

			_, err := b.Send(&tele.User{ID: adminID}, msg, menu, tele.ModeMarkdown)
			if err != nil {
				log.Printf("Failed to send IP notification: %v", err)
			}
		}
	}
}
