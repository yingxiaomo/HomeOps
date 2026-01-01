package openwrt

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yingxiaomo/homeops/pkg/session"
	tele "gopkg.in/telebot.v3"
)

func HandleNetMenu(c tele.Context) error {
	c.Respond()
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("⚡ 快速诊断", "wrt_net_quick"), menu.Data("✍️ 手动测试", "wrt_net_manual")),
		menu.Row(menu.Data("🔙 返回", "wrt_main")),
	)
	return c.Edit("🌐 **网络连接测试**\n请选择测试模式：", menu, tele.ModeMarkdown)
}

func HandleNetQuick(c tele.Context) error {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("📶 Ping 网关", "wrt_net_run_ping_gateway"), menu.Data("📶 Ping 百度", "wrt_net_run_ping_baidu")),
		menu.Row(menu.Data("📶 Ping Google", "wrt_net_run_ping_google"), menu.Data("📶 Ping DNS", "wrt_net_run_ping_dns")),
		menu.Row(menu.Data("📍 Trace Google", "wrt_net_run_trace_google"), menu.Data("🔎 查 Google IP", "wrt_net_run_ns_google")),
		menu.Row(menu.Data("🔙 返回", "wrt_net")),
	)
	return c.Edit("⚡ **快速诊断**\n一键执行常用网络测试：", menu, tele.ModeMarkdown)
}

func HandleNetManual(c tele.Context) error {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("📶 Ping 测试", "wrt_net_ping_ask"), menu.Data("📍 路由追踪", "wrt_net_trace_ask")),
		menu.Row(menu.Data("🔎 DNS 查询", "wrt_net_nslookup_ask"), menu.Data("🌐 HTTP 检测", "wrt_net_curl_ask")),
		menu.Row(menu.Data("🔙 返回", "wrt_net")),
	)
	return c.Edit("✍️ **手动测试**\n请选择工具并输入目标：", menu, tele.ModeMarkdown)
}

func HandleNetRunQuick(c tele.Context, data string) error {
	c.Respond(&tele.CallbackResponse{Text: "正在执行测试..."})

	var cmd, title string
	switch data {
	case "wrt_net_run_ping_gateway":
		gw, _ := SSHExec("ip route | grep default | awk '{print $3}' | head -n 1")
		gw = strings.TrimSpace(gw)
		if gw == "" {
			gw = "192.168.1.1"
		}
		cmd = fmt.Sprintf("ping -c 4 -w 5 %s", gw)
		title = fmt.Sprintf("Ping Gateway (%s)", gw)
	case "wrt_net_run_ping_baidu":
		cmd = "ping -c 4 -w 5 www.baidu.com"
		title = "Ping Baidu"
	case "wrt_net_run_ping_google":
		cmd = "ping -c 4 -w 5 www.google.com"
		title = "Ping Google"
	case "wrt_net_run_ping_dns":
		cmd = "ping -c 4 -w 5 8.8.8.8"
		title = "Ping 8.8.8.8"
	case "wrt_net_run_trace_google":
		cmd = "traceroute -I -m 15 -w 2 -q 1 -n www.google.com 2>/dev/null || traceroute -m 15 -w 2 -q 1 -n www.google.com"
		title = "Trace Google"
	case "wrt_net_run_ns_google":
		cmd = "nslookup www.google.com"
		title = "Nslookup Google"
	}

	c.Edit(fmt.Sprintf("⏳ 正在执行 %s...", title))
	res, _ := SSHExec(cmd)
	if res == "" {
		res = "❌ 执行失败或无输出"
	}
	if len(res) > 3000 {
		res = res[:3000] + "\n...(truncated)"
	}
	res = strings.ReplaceAll(res, "`", "'")

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔙 返回快速诊断", "wrt_net_quick")))
	return c.Edit(fmt.Sprintf("📝 **%s 结果**:\n```\n%s\n```", title, res), menu, tele.ModeMarkdown)
}

func HandleNetPingAsk(c tele.Context) error {
	session.GlobalStore.Set(c.Sender().ID, "wrt_net_state", "ping")
	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("❌ 取消", "wrt_net_manual")))
	return c.Send("📡 请输入要 Ping 的地址/域名：\n(例如: 8.8.8.8 或 google.com)", menu, tele.ForceReply)
}

func HandleNetTraceAsk(c tele.Context) error {
	session.GlobalStore.Set(c.Sender().ID, "wrt_net_state", "trace")
	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("❌ 取消", "wrt_net_manual")))
	return c.Send("📍 请输入要追踪的目标地址：\n(例如: 1.1.1.1)", menu, tele.ForceReply)
}

func HandleNetNslookupAsk(c tele.Context) error {
	session.GlobalStore.Set(c.Sender().ID, "wrt_net_state", "nslookup")
	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("❌ 取消", "wrt_net_manual")))
	return c.Send("🔎 请输入要查询的域名：", menu, tele.ForceReply)
}

func HandleNetCurlAsk(c tele.Context) error {
	session.GlobalStore.Set(c.Sender().ID, "wrt_net_state", "curl")
	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("❌ 取消", "wrt_net_manual")))
	return c.Send("🌐 请输入要检测的 URL：", menu, tele.ForceReply)
}

func HandleNetInput(c tele.Context, state string) error {
	target := c.Text()

	if !regexp.MustCompile(`^[a-zA-Z0-9\.\-\_:/]+$`).MatchString(target) {
		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("❌ 取消", "wrt_net_manual")))
		return c.Send("❌ 检测到非法字符，请重新输入", menu)
	}

	session.GlobalStore.Delete(c.Sender().ID, "wrt_net_state")

	c.Send(fmt.Sprintf("⏳ 正在执行 %s %s...", state, target))

	var cmd string
	switch state {
	case "ping":
		cmd = fmt.Sprintf("ping -c 4 -w 5 %s", target)
	case "trace":
		cmd = fmt.Sprintf("traceroute -I -m 15 -w 2 -q 1 -n %s 2>/dev/null || traceroute -m 15 -w 2 -q 1 -n %s", target, target)
	case "nslookup":
		cmd = fmt.Sprintf("nslookup %s", target)
	case "curl":
		cmd = fmt.Sprintf("curl -I -s -w 'Response Code: %%{http_code}\\nTime: %%{time_total}s\\n' -o /dev/null %s", target)
	}

	res, _ := SSHExec(cmd)
	if res == "" {
		res = "❌ 执行失败或无输出"
	}
	if len(res) > 3000 {
		res = res[:3000] + "\n...(truncated)"
	}
	res = strings.ReplaceAll(res, "`", "'")

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔙 返回手动测试", "wrt_net_manual")))
	return c.Send(fmt.Sprintf("📝 **测试结果**:\n```\n%s\n```", res), menu, tele.ModeMarkdown)
}
