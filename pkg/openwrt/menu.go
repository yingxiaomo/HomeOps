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
	case "adg_toggle":
		return handleAdgToggle(c)
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
	client := NewAdGuardClient()
	
	c.Respond(&tele.CallbackResponse{Text: "正在获取 AdGuard 数据..."})
	
	// Fetch status parallel or seq
	filtering, err1 := client.GetFilteringStatus()
	stats, err2 := client.GetStats()
	
	statusIcon := "🔴"
	statusText := "已禁用"
	if filtering {
		statusIcon = "🟢"
		statusText = "运行中"
	}
	if err1 != nil {
		statusText = fmt.Sprintf("未知 (%v)", err1)
	}

	dnsCount := 0
	blockedCount := 0
	if err2 == nil && stats != nil {
		if v, ok := stats["num_dns_queries"].(float64); ok {
			dnsCount = int(v)
		}
		if v, ok := stats["num_blocked_filtering"].(float64); ok {
			blockedCount = int(v)
		}
	}

	txt := fmt.Sprintf("🛡️ **AdGuard Home**\n"+
		"-------------------\n"+
		"状态: %s %s\n"+
		"查询总数: `%d`\n"+
		"已拦截: `%d`\n",
		statusIcon, statusText, dnsCount, blockedCount)

	menu := &tele.ReplyMarkup{}
	toggleBtn := menu.Data("✅ 开启防护", "adg_toggle")
	if filtering {
		toggleBtn = menu.Data("⛔ 关闭防护", "adg_toggle")
	}
	
	menu.Inline(
		menu.Row(toggleBtn),
		menu.Row(menu.Data("🔄 刷新", "wrt_adg"), menu.Data("🔙 返回", "wrt_main")),
	)
	return c.Edit(txt, menu, tele.ModeMarkdown)
}

func handleAdgToggle(c tele.Context) error {
	client := NewAdGuardClient()
	status, _ := client.GetFilteringStatus()
	
	err := client.SetFiltering(!status)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "操作失败: " + err.Error()})
	}
	
	// Refresh menu
	time.Sleep(500 * time.Millisecond) // Wait for ADG to apply
	return handleAdgMenu(c)
}
