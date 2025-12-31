package openwrt

import (
	"fmt"
	"strings"

	"github.com/yingxiaomo/homeops/pkg/utils"
	tele "gopkg.in/telebot.v3"
)

func HandleDevices(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "获取设备列表中..."})

	adg := NewAdGuardClient()
	leases, err := adg.GetDHCPLeases()
	if err == nil && len(leases) > 0 {
		txt := "📱 **当前联网设备 (ADG DHCP)**\n-------------------\n"
		count := 0
		for _, item := range leases {
			if count >= 100 {
				break
			}
			ip, _ := item["ip"].(string)
			if ip == "" {
				ip = "?"
			}
			name, _ := item["hostname"].(string)
			if name == "" {
				name, _ = item["name"].(string)
			}
			if name == "" {
				name = "(未知)"
			}
			mac, _ := item["mac"].(string)
			macStr := ""
			if mac != "" {
				macStr = fmt.Sprintf(" `[%s]`", mac)
			}
			txt += fmt.Sprintf("• %s (%s)%s\n", utils.EscapeMarkdown(name), utils.EscapeMarkdown(ip), macStr)
			count++
		}

		menu := &tele.ReplyMarkup{}
		menu.Inline(
			menu.Row(menu.Data("🔙 返回", "wrt_main")),
		)
		return c.Edit(txt, menu, tele.ModeMarkdown)
	}

	// 2. Fallback to OpenWrt /tmp/dhcp.leases
	res, err := SSHExec("cat /tmp/dhcp.leases")
	if err == nil && strings.TrimSpace(res) != "" {
		txt := "📱 **当前联网设备 (DHCP)**\n-------------------\n"
		lines := strings.Split(res, "\n")
		hasDev := false
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				ip := parts[2]
				name := parts[3]
				txt += fmt.Sprintf("• %s (%s)\n", utils.EscapeMarkdown(name), utils.EscapeMarkdown(ip))
				hasDev = true
			}
		}
		if !hasDev {
			txt += "没有活跃设备。"
		}

		menu := &tele.ReplyMarkup{}
		menu.Inline(
			menu.Row(menu.Data("🔙 返回", "wrt_main")),
		)
		return c.Edit(txt, menu, tele.ModeMarkdown)
	}

	// 3. Fallback to ARP
	arp, _ := SSHExec("cat /proc/net/arp")
	if strings.TrimSpace(arp) != "" {
		lines := strings.Split(arp, "\n")
		// Check if there are actual entries (skip header)
		if len(lines) > 1 {
			txt := "📱 **当前邻居列表 (ARP)**\n-------------------\n"
			count := 0
			for i, line := range lines {
				if i == 0 {
					continue
				} // Header
				parts := strings.Fields(line)
				if len(parts) >= 4 {
					ip := parts[0]
					mac := parts[3]
					state := ""
					if len(parts) > 5 {
						state = parts[5]
					}
					txt += fmt.Sprintf("• %s `[%s]` %s\n", ip, mac, state)
					count++
				}
			}
			if count > 0 {
				menu := &tele.ReplyMarkup{}
				menu.Inline(
					menu.Row(menu.Data("🔙 返回", "wrt_main")),
				)
				return c.Edit(txt, menu, tele.ModeMarkdown)
			}
		}
	}

	// 4. Fallback to IP Neigh
	neigh, _ := SSHExec("ip neigh show")
	if strings.TrimSpace(neigh) != "" {
		txt := "📱 **当前邻居列表 (IP Neigh)**\n-------------------\n"
		lines := strings.Split(neigh, "\n")
		count := 0
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				ip := parts[0]
				mac := ""
				state := ""
				// format: 192.168.1.135 dev br-lan lladdr 24:18:c6:..:..:.. STALE
				for i, p := range parts {
					if p == "lladdr" && i+1 < len(parts) {
						mac = parts[i+1]
					}
				}
				if len(parts) > 0 {
					state = parts[len(parts)-1]
				}

				macStr := ""
				if mac != "" {
					macStr = fmt.Sprintf(" `[%s]`", mac)
				}
				txt += fmt.Sprintf("• %s%s %s\n", ip, macStr, state)
				count++
			}
		}

		if count > 0 {
			menu := &tele.ReplyMarkup{}
			menu.Inline(
				menu.Row(menu.Data("🔙 返回", "wrt_main")),
			)
			return c.Edit(txt, menu, tele.ModeMarkdown)
		}
	}

	return c.Edit("获取失败或没有活跃设备。", &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{{Text: "🔙 返回", Data: "wrt_main"}},
		},
	})
}
