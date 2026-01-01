package openwrt

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func HandleStatus(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "正在通过 SSH 获取数据..."})
	cmd := "uptime && free -m && [ -f /sys/class/thermal/thermal_zone0/temp ] && cat /sys/class/thermal/thermal_zone0/temp || echo 0"
	res, _ := SSHExec(cmd)
	if res == "" {
		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🔙 返回", "wrt_main")))
		return c.Edit("无法通过 SSH 连接到路由器，请检查配置。", menu)
	}

	lines := strings.Split(res, "\n")
	uptimeInfo := lines[0]
	memTotal := "0"
	memUsed := "0"
	for _, l := range lines {
		if strings.Contains(l, "Mem:") {
			parts := strings.Fields(l)
			if len(parts) >= 3 {
				memTotal = parts[1]
				memUsed = parts[2]
			}
			break
		}
	}
	tempRaw := "0"
	if len(lines) > 0 {
		tempRaw = lines[len(lines)-1]
	}
	temp := "N/A"
	if t, err := fmt.Sscanf(tempRaw, "%d", new(int)); err == nil && t > 0 {
		var val int
		fmt.Sscanf(tempRaw, "%d", &val)
		temp = fmt.Sprintf("%.1f°C", float64(val)/1000.0)
	}

	upSplit := strings.Split(uptimeInfo, "up")
	uptime := ""
	if len(upSplit) > 1 {
		commaSplit := strings.Split(upSplit[1], ",")
		uptime = strings.TrimSpace(commaSplit[0])
	}

	loadSplit := strings.Split(uptimeInfo, "load average:")
	load := ""
	if len(loadSplit) > 1 {
		load = strings.TrimSpace(loadSplit[1])
	}

	txt := fmt.Sprintf("📟 **OpenWrt 状态**\n-------------------\n⏱ 运行时间: %s\n📈 系统负载: %s\n🧠 内存占用: %sMB / %sMB\n🌡 核心温度: %s",
		uptime, load, memUsed, memTotal, temp)

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("🛠 服务管理", "wrt_services_menu"), menu.Data("🧹 清理内存", "wrt_drop_caches")),
		menu.Row(menu.Data("🔙 返回", "wrt_main")),
	)
	return c.Edit(txt, menu, tele.ModeMarkdown)
}

func HandleShowCurrentIPs(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "正在查询 IP..."})
	v4, v6 := GetRouterIPs()

	if v4 == "" && v6 == "" {
		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🔙 返回", "wrt_main")))
		return c.Edit("❌ 无法获取 IP 地址，请检查网络或 SSH 连接。", menu)
	}

	msg := "🏠 **当前公网 IP**\n-------------------\n"
	if v4 != "" {
		msg += fmt.Sprintf("🔴 IPv4: `%s`\n", v4)
	} else {
		msg += "🔴 IPv4: 未检测到\n"
	}
	if v6 != "" {
		msg += fmt.Sprintf("🔵 IPv6: `%s`\n", v6)
	} else {
		msg += "🔵 IPv6: 未检测到\n"
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔙 返回", "wrt_main")))
	return c.Edit(msg, menu, tele.ModeMarkdown)
}

func HandleRebootConfirm(c tele.Context) error {
	c.Respond()
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("✅ 确认重启", "wrt_reboot_do")),
		menu.Row(menu.Data("❌ 取消", "wrt_main")),
	)
	return c.Edit("⚠️ 确认要重启路由器吗？\n重启期间网络将会中断。", menu)
}

func HandleRebootDo(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "指令已发送"})

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔙 返回主菜单", "start_main")))

	c.Edit("🚀 正在重启路由器，请等待网络恢复...", menu)
	go func() {
		SSHExec("reboot")
	}()
	return nil
}

func HandleServicesMenu(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "正在查询服务状态..."})
	services := []string{"network", "firewall", "dnsmasq", "uhttpd"}

	menu := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, svc := range services {
		rows = append(rows, menu.Row(menu.Data(fmt.Sprintf("🔄 重启 %s", svc), "wrt_svc_restart", svc)))
	}
	rows = append(rows, menu.Row(menu.Data("🔙 返回", "wrt_status")))
	menu.Inline(rows...)
	return c.Edit("🛠 **服务管理**\n请选择要操作的服务：", menu, tele.ModeMarkdown)
}

func HandleServiceRestart(c tele.Context) error {
	parts := strings.Split(c.Callback().Data, "|")
	if len(parts) < 2 {
		return c.Respond(&tele.CallbackResponse{Text: "Error: Invalid request"})
	}
	svc := parts[1]
	c.Respond(&tele.CallbackResponse{Text: fmt.Sprintf("正在重启 %s...", svc)})
	c.Edit(fmt.Sprintf("⏳ 正在重启 %s，请稍候...", svc))

	SSHExec(fmt.Sprintf("/etc/init.d/%s restart", svc))

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔙 返回服务列表", "wrt_services_menu")))
	return c.Edit(fmt.Sprintf("✅ %s 重启指令已发送。", svc), menu)
}

func HandleDropCaches(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "正在清理内存..."})
	SSHExec("sync && echo 3 > /proc/sys/vm/drop_caches")
	return HandleStatus(c)
}
