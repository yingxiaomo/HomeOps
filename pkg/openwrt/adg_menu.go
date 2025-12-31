package openwrt

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yingxiaomo/homeops/pkg/session"
	tele "gopkg.in/telebot.v3"
)

func HandleAdgMenu(c tele.Context) error {
	client := NewAdGuardClient()

	if client.BaseURL == "" {
		c.Respond()
		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🔙 返回", "wrt_main")))
		return c.Edit("❌ AdGuard 未配置，请检查 .env 文件。", menu)
	}

	c.Respond(&tele.CallbackResponse{Text: "正在获取 AdGuard 数据..."})

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
		"拦截总数: `%d`\n",
		statusIcon, statusText, dnsCount, blockedCount)

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("🧾 DHCP 租约", "wrt_adg_dhcp"), menu.Data("⚙️ 通用设置", "wrt_adg_general")),
		menu.Row(menu.Data("🧩 DNS 设置", "wrt_adg_dns"), menu.Data("📜 规则与重写", "wrt_adg_rules")),
		menu.Row(menu.Data("🚫 过滤器", "wrt_adg_filters"), menu.Data("♻️ 重启服务", "wrt_adg_restart")),
		menu.Row(menu.Data("🔙 返回", "wrt_main")),
	)
	return c.Edit(txt, menu, tele.ModeMarkdown)
}

func HandleAdgToggle(c tele.Context) error {
	client := NewAdGuardClient()
	filtering, _ := client.GetFilteringStatus()

	newState := !filtering
	err := client.SetFiltering(newState)

	if err != nil {
		c.Respond(&tele.CallbackResponse{Text: fmt.Sprintf("Error: %v", err), ShowAlert: true})
	} else {
		action := "启用"
		if !newState {
			action = "禁用"
		}
		c.Respond(&tele.CallbackResponse{Text: "已" + action})
	}

	return HandleAdgMenu(c)
}

// General Settings
func HandleAdgGeneral(c tele.Context) error {
	client := NewAdGuardClient()
	c.Respond(&tele.CallbackResponse{Text: "获取设置..."})

	ss, _ := client.GetFeatureStatus("/control/safesearch/status")
	pc, _ := client.GetFeatureStatus("/control/parental/status")
	sb, _ := client.GetFeatureStatus("/control/safebrowsing/status")

	qlCfg, _ := client.GetConfig("/control/querylog/config")
	stCfg, _ := client.GetConfig("/control/stats/config")

	qlOn := false
	qlInt := 0.0
	if qlCfg != nil {
		if v, ok := qlCfg["enabled"].(bool); ok {
			qlOn = v
		}
		if v, ok := qlCfg["interval"].(float64); ok {
			qlInt = v
		}
	}

	stOn := false
	stInt := 0.0
	if stCfg != nil {
		if v, ok := stCfg["enabled"].(bool); ok {
			stOn = v
		}
		if v, ok := stCfg["interval"].(float64); ok {
			stInt = v
		}
	}

	fmtDur := func(ms float64) string {
		if ms == 0 {
			return "禁用"
		}
		hrs := ms / 3600000
		if hrs < 24 {
			return fmt.Sprintf("%d小时", int(hrs))
		}
		days := hrs / 24
		return fmt.Sprintf("%d天", int(days))
	}

	menu := &tele.ReplyMarkup{}

	ssIcon := "❌"
	if ss {
		ssIcon = "✅"
	}
	pcIcon := "❌"
	if pc {
		pcIcon = "✅"
	}
	sbIcon := "❌"
	if sb {
		sbIcon = "✅"
	}

	qlText := "禁用"
	if qlOn {
		qlText = fmtDur(qlInt)
	}
	stText := "禁用"
	if stOn {
		stText = fmtDur(stInt)
	}

	menu.Inline(
		menu.Row(menu.Data(fmt.Sprintf("安全搜索: %s", ssIcon), "wrt_adg_gen_toggle_ss|"+strconv.FormatBool(!ss))),
		menu.Row(menu.Data(fmt.Sprintf("家长控制: %s", pcIcon), "wrt_adg_gen_toggle_pc|"+strconv.FormatBool(!pc))),
		menu.Row(menu.Data(fmt.Sprintf("浏览安全: %s", sbIcon), "wrt_adg_gen_toggle_sb|"+strconv.FormatBool(!sb))),
		menu.Row(menu.Data(fmt.Sprintf("查询日志: %s", qlText), "wrt_adg_gen_cycle_log")),
		menu.Row(menu.Data(fmt.Sprintf("统计数据: %s", stText), "wrt_adg_gen_cycle_stats")),
		menu.Row(menu.Data("🔙 返回", "wrt_adg")),
	)
	return c.Edit("⚙️ **通用设置**", menu, tele.ModeMarkdown)
}

func HandleAdgGenToggle(c tele.Context, data string) error {
	parts := strings.Split(data, "|")
	if len(parts) < 2 {
		return c.Respond()
	}
	target := parts[0]
	valStr := parts[1]
	val := (valStr == "true")

	client := NewAdGuardClient()
	c.Respond(&tele.CallbackResponse{Text: "正在切换..."})

	var endpoint string
	if target == "ss" {
		endpoint = "/control/safesearch"
	} else if target == "pc" {
		endpoint = "/control/parental"
	} else if target == "sb" {
		endpoint = "/control/safebrowsing"
	}

	if endpoint != "" {
		client.SetFeatureStatus(endpoint, val)
	}

	return HandleAdgGeneral(c)
}

func HandleAdgGenCycleLog(c tele.Context) error {
	return handleAdgCycle(c, "/control/querylog/config")
}

func HandleAdgGenCycleStats(c tele.Context) error {
	return handleAdgCycle(c, "/control/stats/config")
}

func handleAdgCycle(c tele.Context, endpoint string) error {
	client := NewAdGuardClient()
	c.Respond(&tele.CallbackResponse{Text: "切换时长..."})

	steps := []float64{86400000, 604800000, 2592000000, 7776000000, 0}

	cfg, _ := client.GetConfig(endpoint)
	if cfg == nil {
		return c.Respond()
	}

	currInt := 0.0
	if v, ok := cfg["interval"].(float64); ok {
		currInt = v
	}
	currEn := false
	if v, ok := cfg["enabled"].(bool); ok {
		currEn = v
	}
	if !currEn {
		currInt = 0
	}

	nextInt := steps[0]
	for i, s := range steps {
		if currInt == s {
			nextInt = steps[(i+1)%len(steps)]
			break
		}
	}

	cfg["enabled"] = (nextInt > 0)
	cfg["interval"] = nextInt

	client.SetConfig(endpoint, cfg)
	return HandleAdgGeneral(c)
}

// DNS Settings (Upstream/Bootstrap)
func HandleAdgDns(c tele.Context) error {
	client := NewAdGuardClient()
	c.Respond(&tele.CallbackResponse{Text: "获取 DNS 信息..."})

	info, err := client.GetDNSInfo()
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ 获取失败: %v", err), &tele.ReplyMarkup{
			InlineKeyboard: [][]tele.InlineButton{{{Text: "🔙 返回", Data: "wrt_adg"}}},
		})
	}

	upstream := []string{}
	if v, ok := info["upstream_dns"].([]interface{}); ok {
		for _, u := range v {
			upstream = append(upstream, fmt.Sprint(u))
		}
	}
	bootstrap := []string{}
	if v, ok := info["bootstrap_dns"].([]interface{}); ok {
		for _, b := range v {
			bootstrap = append(bootstrap, fmt.Sprint(b))
		}
	}

	txt := fmt.Sprintf("🧩 **DNS 设置**\n\n"+
		"**上游 DNS**:\n`%s`\n\n"+
		"**Bootstrap DNS**:\n`%s`",
		strings.Join(upstream, "\n"), strings.Join(bootstrap, "\n"))

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("✏️ 编辑上游", "wrt_adg_dns_edit_upstream"), menu.Data("✏️ 编辑 Bootstrap", "wrt_adg_dns_edit_bootstrap")),
		menu.Row(menu.Data("🛠 高级设置", "wrt_adg_dns_advanced")),
		menu.Row(menu.Data("🔙 返回", "wrt_adg")),
	)
	return c.Edit(txt, menu, tele.ModeMarkdown)
}

// DNS Advanced
func HandleAdgDNSAdvanced(c tele.Context) error {
	client := NewAdGuardClient()
	c.Respond(&tele.CallbackResponse{Text: "获取高级设置..."})

	info, err := client.GetDNSInfo()
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ 获取失败: %v", err))
	}

	dnssec := false
	if v, ok := info["dnssec_enabled"].(bool); ok {
		dnssec = v
	}

	ipv6 := false
	if v, ok := info["disable_ipv6"].(bool); ok {
		ipv6 = v
	}

	rl := 0
	if v, ok := info["ratelimit"].(float64); ok {
		rl = int(v)
	}

	cache := 0
	if v, ok := info["cache_size"].(float64); ok {
		cache = int(v)
	}

	bm := "default"
	if v, ok := info["blocking_mode"].(string); ok {
		bm = v
	}

	menu := &tele.ReplyMarkup{}

	secIcon := "❌"
	if dnssec {
		secIcon = "✅"
	}
	v6Icon := "❌"
	if ipv6 {
		v6Icon = "✅"
	}

	menu.Inline(
		menu.Row(menu.Data(fmt.Sprintf("DNSSEC: %s", secIcon), "wrt_adg_dns_toggle_dnssec|"+strconv.FormatBool(!dnssec))),
		menu.Row(menu.Data(fmt.Sprintf("禁用 IPv6: %s", v6Icon), "wrt_adg_dns_toggle_ipv6|"+strconv.FormatBool(!ipv6))),
		menu.Row(menu.Data(fmt.Sprintf("速率限制: %d/s", rl), "wrt_adg_dns_edit_rl")),
		menu.Row(menu.Data(fmt.Sprintf("缓存大小: %d MB", cache/1024/1024), "wrt_adg_dns_edit_cache")),
		menu.Row(menu.Data(fmt.Sprintf("拦截模式: %s", bm), "wrt_adg_dns_cycle_bm")),
		menu.Row(menu.Data("🔙 返回", "wrt_adg_dns")),
	)

	return c.Edit("🛠 **高级 DNS 设置**", menu, tele.ModeMarkdown)
}

func HandleAdgDNSToggle(c tele.Context, data string) error {
	parts := strings.Split(data, "|")
	if len(parts) < 2 {
		return c.Respond()
	}

	target := parts[0]
	val := (parts[1] == "true")

	client := NewAdGuardClient()
	info, _ := client.GetDNSInfo()
	if info == nil {
		return c.Respond()
	}

	if target == "dnssec" {
		info["dnssec_enabled"] = val
	}
	if target == "ipv6" {
		info["disable_ipv6"] = val
	}

	client.SetDNSConfig(info)
	return HandleAdgDNSAdvanced(c)
}

func HandleAdgDnsCycleBM(c tele.Context) error {
	client := NewAdGuardClient()
	info, _ := client.GetDNSInfo()
	if info == nil {
		return c.Respond()
	}

	modes := []string{"default", "null_ip", "custom_ip", "nxdomain"}
	curr := "default"
	if v, ok := info["blocking_mode"].(string); ok {
		curr = v
	}

	next := modes[0]
	for i, m := range modes {
		if curr == m {
			next = modes[(i+1)%len(modes)]
			break
		}
	}

	info["blocking_mode"] = next
	client.SetDNSConfig(info)
	return HandleAdgDNSAdvanced(c)
}

func HandleAdgDhcp(c tele.Context) error {
	client := NewAdGuardClient()
	c.Respond(&tele.CallbackResponse{Text: "读取 DHCP 租约..."})

	leases, err := client.GetDHCPLeases()
	if err != nil || len(leases) == 0 {
		menu := &tele.ReplyMarkup{}
		menu.Inline(
			menu.Row(menu.Data("⚙️ DHCP 设置", "wrt_adg_dhcp_config")),
			menu.Row(menu.Data("📱 使用邻居列表", "wrt_devices")),
			menu.Row(menu.Data("🔙 返回", "wrt_adg")),
		)
		msg := "未获取到租约信息。\n请确保 AdGuard DHCP 已启用或配置正确。"
		return c.Edit(msg, menu)
	}

	txt := "🧾 **当前 DHCP 租约**\n-------------------\n"
	for i, it := range leases {
		if i >= 100 {
			break
		}
		ip := "?"
		if v, ok := it["ip"].(string); ok {
			ip = v
		}
		name := "(未知)"
		if v, ok := it["hostname"].(string); ok {
			name = v
		}
		mac := ""
		if v, ok := it["mac"].(string); ok {
			mac = fmt.Sprintf(" [%s]", v)
		}
		txt += fmt.Sprintf("• %s (%s)%s\n", name, ip, mac)
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("⚙️ DHCP 设置", "wrt_adg_dhcp_config")),
		menu.Row(menu.Data("🔙 返回", "wrt_adg")),
	)
	return c.Edit(txt, menu, tele.ModeMarkdown)
}

func HandleAdgDhcpConfig(c tele.Context) error {
	client := NewAdGuardClient()
	c.Respond(&tele.CallbackResponse{Text: "获取 DHCP 配置..."})

	st, err := client.GetDHCPStatus()
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ 获取失败: %v", err))
	}

	enabled := false
	if v, ok := st["enabled"].(bool); ok {
		enabled = v
	}

	icon := "❌"
	if enabled {
		icon = "✅"
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data(fmt.Sprintf("DHCP 服务: %s", icon), "wrt_adg_dhcp_toggle|"+strconv.FormatBool(!enabled))),
		menu.Row(menu.Data("🔙 返回", "wrt_adg_dhcp")),
	)
	return c.Edit("⚙️ **DHCP 设置**", menu, tele.ModeMarkdown)
}

func HandleAdgDhcpToggle(c tele.Context, data string) error {
	val := (data == "true")
	client := NewAdGuardClient()
	st, _ := client.GetDHCPStatus()
	if st == nil {
		return c.Respond()
	}

	st["enabled"] = val
	// API requires v4/v6 fields to be present usually
	client.SetDHCPConfig(st)
	return HandleAdgDhcpConfig(c)
}

// Rules and Rewrites
func HandleAdgRules(c tele.Context) error {
	client := NewAdGuardClient()
	c.Respond(&tele.CallbackResponse{Text: "获取规则..."})

	status, err := client.GetFiltering()
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ 获取失败: %v", err))
	}

	rules := []string{}
	if v, ok := status["user_rules"].([]interface{}); ok {
		for _, r := range v {
			rules = append(rules, fmt.Sprint(r))
		}
	}

	txt := "📜 **自定义规则**\n-------------------\n"
	if len(rules) == 0 {
		txt += "(无)"
	} else {
		for _, r := range rules {
			txt += fmt.Sprintf("`%s`\n", r)
		}
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("➕ 添加/删除规则", "wrt_adg_rules_edit")),
		menu.Row(menu.Data("🔙 返回", "wrt_adg")),
	)
	return c.Edit(txt, menu, tele.ModeMarkdown)
}

func HandleAdgFilters(c tele.Context) error {
	client := NewAdGuardClient()
	c.Respond(&tele.CallbackResponse{Text: "获取过滤器..."})

	status, err := client.GetFiltering()
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ 获取失败: %v", err))
	}

	txt := "🚫 **过滤器列表**\n-------------------\n"
	if v, ok := status["filters"].([]interface{}); ok {
		for _, f := range v {
			if fm, ok := f.(map[string]interface{}); ok {
				name := fm["name"]
				enabled := fm["enabled"]
				count := 0
				if c, ok := fm["rules_count"].(float64); ok {
					count = int(c)
				}
				icon := "🔴"
				if enabled == true {
					icon = "🟢"
				}
				txt += fmt.Sprintf("%s %s (%d)\n", icon, name, count)
			}
		}
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("➕ 添加列表", "wrt_adg_filter_add"), menu.Data("➖ 删除列表", "wrt_adg_filter_del")),
		menu.Row(menu.Data("🔙 返回", "wrt_adg")),
	)
	return c.Edit(txt, menu, tele.ModeMarkdown)
}

func HandleAdgRestart(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "正在重启 AdGuard..."})
	SSHExec("/etc/init.d/AdGuardHome restart || /etc/init.d/adguardhome restart")
	return c.Send("✅ AdGuard 服务已重启。")
}

// Wizard Handlers
func HandleAdgWizardInput(c tele.Context, state map[string]interface{}) bool {
	mode, ok := state["mode"].(string)
	if !ok {
		return false
	}
	text := c.Text()
	client := NewAdGuardClient()
	userID := c.Sender().ID

	switch mode {
	case "set_upstreams":
		lines := strings.Split(text, "\n")
		cfg, _ := client.GetDNSInfo()
		if cfg != nil {
			cfg["upstream_dns"] = lines
			client.SetDNSConfig(cfg)
			c.Send("✅ 已更新上游 DNS。")
		} else {
			c.Send("❌ 更新失败。")
		}
	case "set_bootstrap":
		lines := strings.Split(text, "\n")
		cfg, _ := client.GetDNSInfo()
		if cfg != nil {
			cfg["bootstrap_dns"] = lines
			client.SetDNSConfig(cfg)
			c.Send("✅ 已更新 Bootstrap DNS。")
		} else {
			c.Send("❌ 更新失败。")
		}
	case "edit_rules":
		rule := strings.TrimSpace(text)
		status, _ := client.GetFiltering()
		if status != nil {
			rules := []string{}
			if v, ok := status["user_rules"].([]interface{}); ok {
				for _, r := range v {
					rules = append(rules, fmt.Sprint(r))
				}
			}
			newRules := []string{}
			deleted := false
			for _, r := range rules {
				if r == rule {
					deleted = true
				} else {
					newRules = append(newRules, r)
				}
			}
			msg := ""
			if deleted {
				client.SetRules(newRules)
				msg = fmt.Sprintf("✅ 已删除规则: `%s`", rule)
			} else {
				newRules = append(newRules, rule)
				client.SetRules(newRules)
				msg = fmt.Sprintf("✅ 已添加规则: `%s`", rule)
			}
			c.Send(msg, tele.ModeMarkdown)
		}
	case "set_ratelimit":
		val, err := strconv.Atoi(text)
		if err == nil {
			cfg, _ := client.GetDNSInfo()
			if cfg != nil {
				cfg["ratelimit"] = val
				client.SetDNSConfig(cfg)
				c.Send(fmt.Sprintf("✅ 速率限制已设置为 %d/s。", val))
			}
		} else {
			c.Send("❌ 无效的数字。")
		}
	case "set_cache":
		val, err := strconv.Atoi(text)
		if err == nil {
			cfg, _ := client.GetDNSInfo()
			if cfg != nil {
				cfg["cache_size"] = val * 1024 * 1024
				client.SetDNSConfig(cfg)
				c.Send(fmt.Sprintf("✅ 缓存大小已设置为 %d MB。", val))
			}
		} else {
			c.Send("❌ 无效的数字。")
		}
	case "add_filter":
		parts := strings.SplitN(text, " ", 2)
		if len(parts) == 2 {
			err := client.AddFilter(parts[0], parts[1], false)
			if err == nil {
				c.Send(fmt.Sprintf("✅ 已添加过滤器: %s", parts[0]))
			} else {
				c.Send(fmt.Sprintf("❌ 添加失败: %v", err))
			}
		} else {
			c.Send("❌ 格式错误，请使用: 名称 URL")
		}
	case "del_filter":
		url := strings.TrimSpace(text)
		err := client.RemoveFilter(url, false)
		if err == nil {
			c.Send("✅ 已删除过滤器。")
		} else {
			c.Send(fmt.Sprintf("❌ 删除失败: %v", err))
		}
	default:
		return false
	}

	session.GlobalStore.Delete(userID, "adg_wizard")

	backBtn := "wrt_adg"
	switch mode {
	case "set_upstreams", "set_bootstrap":
		backBtn = "wrt_adg_dns"
	case "set_ratelimit", "set_cache":
		backBtn = "wrt_adg_dns_advanced"
	case "edit_rules":
		backBtn = "wrt_adg_rules"
	case "add_filter", "del_filter":
		backBtn = "wrt_adg_filters"
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔙 返回", backBtn)))
	c.Send("✅ 操作已完成。", menu)

	return true
}

func HandleAdgStartWizard(c tele.Context, mode string, msg string) error {
	session.GlobalStore.Set(c.Sender().ID, "adg_wizard", map[string]interface{}{
		"mode": mode,
	})

	// Determine cancel button destination
	cancelBtn := "wrt_adg"
	switch mode {
	case "set_upstreams", "set_bootstrap":
		cancelBtn = "wrt_adg_dns"
	case "set_ratelimit", "set_cache":
		cancelBtn = "wrt_adg_dns_advanced"
	case "edit_rules":
		cancelBtn = "wrt_adg_rules"
	case "add_filter", "del_filter":
		cancelBtn = "wrt_adg_filters"
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("❌ 取消", cancelBtn)))

	return c.Send(msg, menu, &tele.ReplyMarkup{ForceReply: true})
}
