package openwrt

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

func HandleWrtMain(c tele.Context) error {
	c.Respond()
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("📈 系统状态", "wrt_status"), menu.Data("🏠 当前 IP", "wrt_show_current_ips")),
		menu.Row(menu.Data("📱 联网设备", "wrt_devices"), menu.Data("🌐 网络工具", "wrt_net")),
		menu.Row(menu.Data("📜 运行脚本", "wrt_scripts_list"), menu.Data("🔥 防火墙", "wrt_fw_menu")),
		menu.Row(menu.Data("🛡️ AdGuard", "wrt_adg"), menu.Data("🔄 重启系统", "wrt_reboot_confirm")),
		menu.Row(menu.Data("🤖 AI 分析日志", "wrt_ai_analyze"), menu.Data("🔙 返回", "start_main")),
	)
	return c.EditOrSend("📡 **OpenWrt 管理面板**\n请选择功能：", menu, tele.ModeMarkdown)
}

func HandleCallback(c tele.Context, data string) error {
	if strings.HasPrefix(data, "wrt_net_run_") {
		return HandleNetRunQuick(c, data)
	}
	if strings.HasPrefix(data, "wrt_run_") {
		if strings.HasPrefix(data, "wrt_run_script") {
			return HandleRunScript(c)
		}
	}

	if strings.HasPrefix(data, "wrt_fw_del") {
		return HandleFwDel(c)
	}
	if strings.HasPrefix(data, "wrt_fw_rename_") {
		return HandleFwRename(c)
	}
	if strings.HasPrefix(data, "wrt_svc_restart") {
		return HandleServiceRestart(c)
	}

	if strings.HasPrefix(data, "wrt_fw_wiz_proto") {
		return HandleFwWizardProto(c)
	}
	if strings.HasPrefix(data, "wrt_fw_wiz_target") {
		return HandleFwWizardTarget(c)
	}

	if strings.HasPrefix(data, "wrt_adg_gen_toggle_") {
		return HandleAdgGenToggle(c, strings.TrimPrefix(data, "wrt_adg_gen_toggle_"))
	}
	if strings.HasPrefix(data, "wrt_adg_dns_toggle_") {
		return HandleAdgDNSToggle(c, strings.TrimPrefix(data, "wrt_adg_dns_toggle_"))
	}
	if strings.HasPrefix(data, "wrt_adg_dhcp_toggle") {
		parts := strings.Split(data, "|")
		if len(parts) == 2 {
			return HandleAdgDhcpToggle(c, parts[1])
		}
		return c.Respond()
	}

	switch data {
	case "wrt_main", "wrt_exit":
		return HandleWrtMain(c)
	case "wrt_status":
		return HandleStatus(c)
	case "wrt_show_current_ips":
		return HandleShowCurrentIPs(c)
	case "wrt_devices":
		return HandleDevices(c)
	case "wrt_net":
		return HandleNetMenu(c)
	case "wrt_net_quick":
		return HandleNetQuick(c)
	case "wrt_net_manual":
		return HandleNetManual(c)
	case "wrt_net_ping_ask":
		return HandleNetPingAsk(c)
	case "wrt_net_trace_ask":
		return HandleNetTraceAsk(c)
	case "wrt_net_nslookup_ask":
		return HandleNetNslookupAsk(c)
	case "wrt_net_curl_ask":
		return HandleNetCurlAsk(c)
	case "wrt_scripts_list":
		return HandleScriptsList(c)
	case "wrt_fw_menu":
		return HandleFwMenu(c)
	case "wrt_fw_list_redirects":
		return HandleFwListRedirects(c)
	case "wrt_fw_list_rules":
		return HandleFwListRules(c)
	case "wrt_fw_list_all":
		return HandleFwListAll(c)
	case "wrt_fw_add_redirect_start":
		return HandleFwAddRedirectStart(c)
	case "wrt_fw_add_rule_start":
		return HandleFwAddRuleStart(c)
	case "wrt_adg":
		return HandleAdgMenu(c)
	case "wrt_adg_toggle":
		return HandleAdgToggle(c)
	case "wrt_adg_general":
		return HandleAdgGeneral(c)
	case "wrt_adg_gen_cycle_log":
		return HandleAdgGenCycleLog(c)
	case "wrt_adg_gen_cycle_stats":
		return HandleAdgGenCycleStats(c)
	case "wrt_adg_dns":
		return HandleAdgDns(c)
	case "wrt_adg_dns_advanced":
		return HandleAdgDNSAdvanced(c)
	case "wrt_adg_dns_edit_upstream":
		return HandleAdgStartWizard(c, "set_upstreams", "请输入新的上游 DNS (每行一个):")
	case "wrt_adg_dns_edit_bootstrap":
		return HandleAdgStartWizard(c, "set_bootstrap", "请输入新的 Bootstrap DNS (每行一个):")
	case "wrt_adg_dns_edit_rl":
		return HandleAdgStartWizard(c, "set_ratelimit", "请输入速率限制 (次/秒):")
	case "wrt_adg_dns_edit_cache":
		return HandleAdgStartWizard(c, "set_cache", "请输入缓存大小 (MB):")
	case "wrt_adg_dns_cycle_bm":
		return HandleAdgDnsCycleBM(c)
	case "wrt_adg_dhcp":
		return HandleAdgDhcp(c)
	case "wrt_adg_dhcp_config":
		return HandleAdgDhcpConfig(c)
	case "wrt_adg_rules":
		return HandleAdgRules(c)
	case "wrt_adg_rules_edit":
		return HandleAdgStartWizard(c, "edit_rules", "请输入要添加或删除的规则 (精确匹配删除):")
	case "wrt_adg_filters":
		return HandleAdgFilters(c)
	case "wrt_adg_filter_add":
		return HandleAdgStartWizard(c, "add_filter", "请输入过滤器 名称 和 URL (空格分隔):")
	case "wrt_adg_filter_del":
		return HandleAdgStartWizard(c, "del_filter", "请输入要删除的过滤器 URL:")
	case "wrt_adg_filter_refresh":
		return HandleAdgFilters(c)
	case "wrt_adg_restart":
		return HandleAdgRestart(c)
	case "wrt_ai_analyze":
		return HandleAIAnalyze(c)
	case "wrt_reboot_confirm":
		return HandleRebootConfirm(c)
	case "wrt_reboot_do":
		return HandleRebootDo(c)
	case "wrt_services_menu":
		return HandleServicesMenu(c)
	case "wrt_drop_caches":
		return HandleDropCaches(c)
	}
	return c.Respond()
}
