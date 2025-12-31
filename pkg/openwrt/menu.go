package openwrt

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

// HandleWrtMain shows the main menu
func HandleWrtMain(c tele.Context) error {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("📱 设备列表", "wrt_devices"), menu.Data("🌐 网络工具", "wrt_net")),
		menu.Row(menu.Data("🛡️ AdGuard", "wrt_adg"), menu.Data("🔥 防火墙", "wrt_firewall")),
		menu.Row(menu.Data("📊 系统状态", "wrt_status"), menu.Data("⚙️ 服务管理", "wrt_services")),
	)
	return c.EditOrSend("📡 **OpenWrt 管理面板**\n请选择功能：", menu, tele.ModeMarkdown)
}

// HandleCallback routes all wrt_ callbacks
func HandleCallback(c tele.Context, data string) error {
	switch data {
	case "wrt_main":
		return HandleWrtMain(c)
	case "wrt_status":
		return handleStatus(c)
	case "wrt_devices":
		return handleDevices(c)
	case "wrt_net":
		return handleNetMenu(c)
	case "wrt_adg":
		return handleAdgMenu(c)
	// Add more cases
	}
	return c.Respond()
}

func handleStatus(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "正在获取状态..."})
	status := GetSystemStatus()
	
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("🔄 刷新", "wrt_status"), menu.Data("🔙 返回", "wrt_main")),
	)
	return c.Edit(fmt.Sprintf("📊 **系统状态**\n```\n%s\n```", status), menu, tele.ModeMarkdown)
}

func handleDevices(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "扫描设备中..."})
	
	// TODO: Implement actual parsing of dhcp.leases and arp
	res, _ := SSHExec("cat /tmp/dhcp.leases")
	
	txt := "📱 **设备列表**\n-------------------\n"
	if res == "" {
		txt += "暂无 DHCP 记录。"
	} else {
		lines := strings.Split(res, "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				// time mac ip name
				txt += fmt.Sprintf("• %s (%s)\n", parts[3], parts[2])
			}
		}
	}
	
	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔙 返回", "wrt_main")))
	return c.Edit(txt, menu, tele.ModeMarkdown)
}

func handleNetMenu(c tele.Context) error {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("📡 Ping", "wrt_ping"), menu.Data("🛣️ Trace", "wrt_trace")),
		menu.Row(menu.Data("🔎 Nslookup", "wrt_nslookup"), menu.Data("🌐 Curl", "wrt_curl")),
		menu.Row(menu.Data("🔙 返回", "wrt_main")),
	)
	return c.Edit("🌐 **网络工具箱**", menu)
}

func handleAdgMenu(c tele.Context) error {
	// Simple placeholder for ADG
	txt := "🛡️ **AdGuard Home**\n目前仅支持查看状态 (Go版开发中)。"
	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔙 返回", "wrt_main")))
	return c.Edit(txt, menu, tele.ModeMarkdown)
}
