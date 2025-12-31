package openwrt

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/yingxiaomo/homeops/pkg/session"
	"github.com/yingxiaomo/homeops/pkg/utils"
	tele "gopkg.in/telebot.v3"
)

func parseUCIFirewall(output string, prefix string) map[string]map[string]string {
	rules := make(map[string]map[string]string)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "'")

		keyParts := strings.Split(key, ".")
		if len(keyParts) < 2 {
			continue
		}
		section := keyParts[1]
		if prefix != "" && !strings.HasPrefix(section, prefix) {
			continue
		}

		if _, ok := rules[section]; !ok {
			rules[section] = make(map[string]string)
		}

		if len(keyParts) == 3 {
			rules[section][keyParts[2]] = value
		} else {
			rules[section]["_type"] = value
		}
	}
	return rules
}

func HandleFwMenu(c tele.Context) error {
	session.GlobalStore.Delete(c.Sender().ID, "fw_wizard")
	c.Respond()
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("🔀 端口转发列表", "wrt_fw_list_redirects"), menu.Data("➕ 添加转发", "wrt_fw_add_redirect_start")),
		menu.Row(menu.Data("🛡️ 通信规则列表", "wrt_fw_list_rules"), menu.Data("➕ 添加规则", "wrt_fw_add_rule_start")),
		menu.Row(menu.Data("📋 显示全部", "wrt_fw_list_all")),
		menu.Row(menu.Data("🔙 返回", "wrt_main")),
	)
	return c.Edit("🔥 防火墙管理\n仅显示前缀为 `homeops_` 的规则。", menu, tele.ModeMarkdown)
}

func HandleFwListRedirects(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "读取配置中..."})
	res, _ := SSHExec("uci show firewall")
	rules := parseUCIFirewall(res, "")

	txt := "🔀 **端口转发 (Redirects)**\n-------------------\n"
	menu := &tele.ReplyMarkup{}
	var rows []tele.Row

	count := 0
	for sec, data := range rules {
		if data["_type"] != "redirect" {
			continue
		}
		if !strings.HasPrefix(sec, "homeops_") {
			continue
		}
		count++
		name := strings.TrimPrefix(sec, "homeops_")
		srcDport := data["src_dport"]
		if srcDport == "" {
			srcDport = "?"
		}
		destIp := data["dest_ip"]
		if destIp == "" {
			destIp = "?"
		}
		destPort := data["dest_port"]
		if destPort == "" {
			destPort = srcDport
		}
		proto := data["proto"]
		if proto == "" {
			proto = "tcp"
		}

		txt += fmt.Sprintf("🔹 `%s`: %s :%s ➝ %s:%s\n", name, utils.EscapeMarkdown(strings.ToUpper(proto)), utils.EscapeMarkdown(srcDport), utils.EscapeMarkdown(destIp), utils.EscapeMarkdown(destPort))
		rows = append(rows, menu.Row(menu.Data(fmt.Sprintf("🗑️ 删除 %s", name), "wrt_fw_del", sec)))
	}
	if count == 0 {
		txt += "无记录。"
	}

	rows = append(rows, menu.Row(menu.Data("🔙 返回", "wrt_fw_menu")))
	menu.Inline(rows...)
	return c.Edit(txt, menu, tele.ModeMarkdown)
}

func HandleFwListRules(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "读取配置中..."})
	res, _ := SSHExec("uci show firewall")
	rules := parseUCIFirewall(res, "")

	txt := "🛡️ **通信规则 (Rules)**\n-------------------\n"
	menu := &tele.ReplyMarkup{}
	var rows []tele.Row

	count := 0
	for sec, data := range rules {
		if data["_type"] != "rule" {
			continue
		}
		if !strings.HasPrefix(sec, "homeops_") {
			continue
		}
		count++
		name := strings.TrimPrefix(sec, "homeops_")
		src := data["src"]
		if src == "" {
			src = "*"
		}
		dest := data["dest"]
		if dest == "" {
			dest = "*"
		}
		destPort := data["dest_port"]
		if destPort == "" {
			destPort = "All"
		}
		target := data["target"]
		if target == "" {
			target = "?"
		}

		txt += fmt.Sprintf("🔸 `%s`: %s➝%s :%s (%s)\n", name, utils.EscapeMarkdown(src), utils.EscapeMarkdown(dest), utils.EscapeMarkdown(destPort), utils.EscapeMarkdown(target))
		rows = append(rows, menu.Row(menu.Data(fmt.Sprintf("🗑️ 删除 %s", name), "wrt_fw_del", sec)))
	}
	if count == 0 {
		txt += "无记录。"
	}

	rows = append(rows, menu.Row(menu.Data("🔙 返回", "wrt_fw_menu")))
	menu.Inline(rows...)
	return c.Edit(txt, menu, tele.ModeMarkdown)
}

func HandleFwListAll(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "读取全部规则..."})
	res, _ := SSHExec("uci show firewall")
	rules := parseUCIFirewall(res, "")

	txt := "📋 **全部防火墙配置**\n-------------------\n"
	menu := &tele.ReplyMarkup{}
	var rows []tele.Row
	var redirects []string
	var ruleLines []string

	for sec, data := range rules {
		t := data["_type"]
		tag := "系统"
		if strings.HasPrefix(sec, "homeops_") {
			tag = "HomeOps"
		}

		name := data["name"]
		if name == "" {
			name = sec
		}

		if t == "redirect" {
			srcDport := data["src_dport"]
			if srcDport == "" {
				srcDport = "?"
			}
			destIp := data["dest_ip"]
			if destIp == "" {
				destIp = "?"
			}
			destPort := data["dest_port"]
			if destPort == "" {
				destPort = srcDport
			}
			proto := data["proto"]
			if proto == "" {
				proto = "tcp"
			}

			redirects = append(redirects, fmt.Sprintf("🔀 [%s] `%s`: %s :%s ➝ %s:%s (`%s`)", tag, name, utils.EscapeMarkdown(strings.ToUpper(proto)), utils.EscapeMarkdown(srcDport), utils.EscapeMarkdown(destIp), utils.EscapeMarkdown(destPort), sec))
			if tag == "系统" {
				rows = append(rows, menu.Row(menu.Data(fmt.Sprintf("迁移为可管理: %s", name), fmt.Sprintf("wrt_fw_rename_%s", sec))))
			}
		} else if t == "rule" {
			src := data["src"]
			if src == "" {
				src = "*"
			}
			dest := data["dest"]
			if dest == "" {
				dest = "*"
			}
			destPort := data["dest_port"]
			if destPort == "" {
				destPort = "All"
			}
			target := data["target"]
			if target == "" {
				target = "?"
			}

			ruleLines = append(ruleLines, fmt.Sprintf("🛡️ [%s] `%s`: %s➝%s :%s (%s) (`%s`)", tag, name, utils.EscapeMarkdown(src), utils.EscapeMarkdown(dest), utils.EscapeMarkdown(destPort), utils.EscapeMarkdown(target), sec))
			if tag == "系统" {
				rows = append(rows, menu.Row(menu.Data(fmt.Sprintf("迁移为可管理: %s", name), fmt.Sprintf("wrt_fw_rename_%s", sec))))
			}
		}
	}

	if len(redirects) > 0 {
		txt += "Redirects:\n" + strings.Join(redirects, "\n") + "\n"
	}
	if len(ruleLines) > 0 {
		txt += "Rules:\n" + strings.Join(ruleLines, "\n")
	}
	if len(redirects) == 0 && len(ruleLines) == 0 {
		txt += "无记录。"
	}

	rows = append(rows, menu.Row(menu.Data("🔙 返回", "wrt_fw_menu")))
	menu.Inline(rows...)

	if len(txt) > 4000 {
		txt = txt[:4000] + "\n...(truncated)"
	}
	return c.Edit(txt, menu, tele.ModeMarkdown)
}

func HandleFwDel(c tele.Context) error {
	parts := strings.Split(c.Callback().Data, "|")
	if len(parts) < 2 {
		return c.Respond(&tele.CallbackResponse{Text: "Error: Missing section"})
	}
	sec := parts[1]

	c.Respond(&tele.CallbackResponse{Text: "正在删除..."})
	cmd := fmt.Sprintf("uci delete firewall.%s && uci commit firewall && /etc/init.d/firewall reload", sec)
	SSHExec(cmd)

	return HandleFwMenu(c)
}

func HandleFwRename(c tele.Context) error {
	sec := strings.TrimPrefix(c.Callback().Data, "wrt_fw_rename_")
	c.Respond(&tele.CallbackResponse{Text: "正在迁移为可管理..."})

	res, _ := SSHExec("uci show firewall")
	rules := parseUCIFirewall(res, "")
	if data, ok := rules[sec]; ok {
		t := data["_type"]
		rawName := data["name"]
		if rawName == "" {
			rawName = sec
		}

		base := strings.ToLower(rawName)
		base = regexp.MustCompile(`[^a-z0-9_]`).ReplaceAllString(base, "_")
		if base == "" {
			if t == "redirect" {
				base = "redirect"
			} else {
				base = "rule"
			}
		}

		idx := "0"
		matches := regexp.MustCompile(`\[(\d+)\]`).FindStringSubmatch(sec)
		if len(matches) > 1 {
			idx = matches[1]
		}

		newSec := fmt.Sprintf("homeops_%s", base)
		if _, exists := rules[newSec]; exists {
			newSec = fmt.Sprintf("homeops_%s_%s", base, idx)
		}

		cmd := fmt.Sprintf("uci rename firewall.%s=%s && uci commit firewall && /etc/init.d/firewall reload", sec, newSec)
		SSHExec(cmd)

		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("📋 返回全部", "wrt_fw_list_all")))
		return c.Edit(fmt.Sprintf("✅ 已迁移为可管理: %s", newSec), menu)
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔙 返回", "wrt_fw_list_all")))
	return c.Edit(fmt.Sprintf("未找到段: %s", sec), menu)
}

type FwWizardState struct {
	Type string            `json:"type"`
	Step string            `json:"step"`
	Data map[string]string `json:"data"`
}

func HandleFwAddRedirectStart(c tele.Context) error {
	c.Respond()
	state := FwWizardState{
		Type: "redirect",
		Step: "name",
		Data: make(map[string]string),
	}
	session.GlobalStore.Set(c.Sender().ID, "fw_wizard", state)

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("取消", "wrt_fw_menu")))
	return c.Send("➕ **添加端口转发 - 第 1/5 步**\n请输入规则名称 (如: web):", menu, tele.ModeMarkdown, tele.ForceReply)
}

func HandleFwAddRuleStart(c tele.Context) error {
	c.Respond()
	state := FwWizardState{
		Type: "rule",
		Step: "name",
		Data: make(map[string]string),
	}
	session.GlobalStore.Set(c.Sender().ID, "fw_wizard", state)

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("取消", "wrt_fw_menu")))
	return c.Send("➕ **添加通信规则 - 第 1/5 步**\n请输入规则名称 (如: allow_ssh):", menu, tele.ModeMarkdown, tele.ForceReply)
}

func HandleFwWizardInput(c tele.Context, text string) error {
	userID := c.Sender().ID
	val := session.GlobalStore.Get(userID, "fw_wizard")
	if val == nil {
		return nil
	}
	state, ok := val.(FwWizardState)
	if !ok {
		return nil
	}

	text = strings.TrimSpace(text)
	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("取消", "wrt_fw_menu")))

	if state.Step == "name" {
		if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(text) {
			return c.Send("❌ 名称只能包含字母、数字和下划线。请重新输入:", menu, tele.ModeMarkdown, tele.ForceReply)
		}
	}

	switch state.Type {
	case "redirect":
		switch state.Step {
		case "name":
			state.Data["name"] = text
			state.Step = "ext_port"
			session.GlobalStore.Set(userID, "fw_wizard", state)
			return c.Send("➕ **第 2/5 步**\n请输入外部端口 (Src Dport):", menu, tele.ModeMarkdown, tele.ForceReply)
		case "ext_port":
			if _, err := strconv.Atoi(text); err != nil {
				return c.Send("❌ 端口必须是数字。请重新输入:", menu, tele.ModeMarkdown, tele.ForceReply)
			}
			state.Data["ext_port"] = text
			state.Step = "int_ip"
			session.GlobalStore.Set(userID, "fw_wizard", state)
			return c.Send("➕ **第 3/5 步**\n请输入内部 IP (Dest IP):", menu, tele.ModeMarkdown, tele.ForceReply)
		case "int_ip":
			state.Data["int_ip"] = text
			state.Step = "int_port"
			session.GlobalStore.Set(userID, "fw_wizard", state)
			return c.Send("➕ **第 4/5 步**\n请输入内部端口 (Dest Port):", menu, tele.ModeMarkdown, tele.ForceReply)
		case "int_port":
			if _, err := strconv.Atoi(text); err != nil {
				return c.Send("❌ 端口必须是数字。请重新输入:", menu, tele.ModeMarkdown, tele.ForceReply)
			}
			state.Data["int_port"] = text
			state.Step = "proto"
			session.GlobalStore.Set(userID, "fw_wizard", state)

			protoMenu := &tele.ReplyMarkup{}
			protoMenu.Inline(
				protoMenu.Row(protoMenu.Data("TCP", "wrt_fw_wiz_proto", "tcp"), protoMenu.Data("UDP", "wrt_fw_wiz_proto", "udp")),
				protoMenu.Row(protoMenu.Data("TCP+UDP", "wrt_fw_wiz_proto", "tcp udp")),
				protoMenu.Row(protoMenu.Data("取消", "wrt_fw_menu")),
			)
			return c.Send("➕ **第 5/5 步**\n请选择协议:", protoMenu, tele.ModeMarkdown)
		}
	case "rule":
		switch state.Step {
		case "name":
			state.Data["name"] = text
			state.Step = "src"
			session.GlobalStore.Set(userID, "fw_wizard", state)
			return c.Send("➕ **第 2/5 步**\n请输入源区域 (Src, 如: wan):", menu, tele.ModeMarkdown, tele.ForceReply)
		case "src":
			state.Data["src"] = text
			state.Step = "dest"
			session.GlobalStore.Set(userID, "fw_wizard", state)
			return c.Send("➕ **第 3/5 步**\n请输入目标区域 (Dest, 如: lan):", menu, tele.ModeMarkdown, tele.ForceReply)
		case "dest":
			state.Data["dest"] = text
			state.Step = "dest_port"
			session.GlobalStore.Set(userID, "fw_wizard", state)
			return c.Send("➕ **第 4/5 步**\n请输入目标端口 (Dest Port, 留空表示全部):", menu, tele.ModeMarkdown, tele.ForceReply)
		case "dest_port":
			state.Data["dest_port"] = text
			state.Step = "target"
			session.GlobalStore.Set(userID, "fw_wizard", state)

			targetMenu := &tele.ReplyMarkup{}
			targetMenu.Inline(
				targetMenu.Row(targetMenu.Data("ACCEPT (允许)", "wrt_fw_wiz_target", "ACCEPT"), targetMenu.Data("DROP (丢弃)", "wrt_fw_wiz_target", "DROP")),
				targetMenu.Row(targetMenu.Data("REJECT (拒绝)", "wrt_fw_wiz_target", "REJECT")),
				targetMenu.Row(targetMenu.Data("取消", "wrt_fw_menu")),
			)
			return c.Send("➕ **第 5/5 步**\n请选择动作 (Target):", targetMenu, tele.ModeMarkdown)
		}
	}
	return nil
}

func HandleFwWizardProto(c tele.Context) error {
	userID := c.Sender().ID
	val := session.GlobalStore.Get(userID, "fw_wizard")
	if val == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Session expired"})
	}
	state, ok := val.(FwWizardState)
	if !ok || state.Type != "redirect" || state.Step != "proto" {
		return c.Respond()
	}

	parts := strings.Split(c.Callback().Data, "|")
	if len(parts) < 2 {
		return c.Respond()
	}
	proto := parts[1]
	state.Data["proto"] = proto

	c.Respond(&tele.CallbackResponse{Text: "正在提交..."})
	c.Edit(fmt.Sprintf("⏳ 正在添加端口转发 %s...", state.Data["name"]))

	name := state.Data["name"]
	sec := fmt.Sprintf("homeops_%s", name)
	cmds := []string{
		fmt.Sprintf("uci set firewall.%s=redirect", sec),
		fmt.Sprintf("uci set firewall.%s.name='%s'", sec, name),
		fmt.Sprintf("uci set firewall.%s.src='wan'", sec),
		fmt.Sprintf("uci set firewall.%s.src_dport='%s'", sec, state.Data["ext_port"]),
		fmt.Sprintf("uci set firewall.%s.dest='lan'", sec),
		fmt.Sprintf("uci set firewall.%s.dest_ip='%s'", sec, state.Data["int_ip"]),
		fmt.Sprintf("uci set firewall.%s.dest_port='%s'", sec, state.Data["int_port"]),
		fmt.Sprintf("uci set firewall.%s.proto='%s'", sec, proto),
		fmt.Sprintf("uci set firewall.%s.target='DNAT'", sec),
		"uci commit firewall",
		"/etc/init.d/firewall reload",
	}

	SSHExec(strings.Join(cmds, " && "))
	session.GlobalStore.Delete(userID, "fw_wizard")

	return HandleFwMenu(c)
}

func HandleFwWizardTarget(c tele.Context) error {
	userID := c.Sender().ID
	val := session.GlobalStore.Get(userID, "fw_wizard")
	if val == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Session expired"})
	}
	state, ok := val.(FwWizardState)
	if !ok || state.Type != "rule" || state.Step != "target" {
		return c.Respond()
	}

	parts := strings.Split(c.Callback().Data, "|")
	if len(parts) < 2 {
		return c.Respond()
	}
	target := parts[1]
	state.Data["target"] = target

	c.Respond(&tele.CallbackResponse{Text: "正在提交..."})
	c.Edit(fmt.Sprintf("⏳ 正在添加通信规则 %s...", state.Data["name"]))

	name := state.Data["name"]
	sec := fmt.Sprintf("homeops_%s", name)
	cmds := []string{
		fmt.Sprintf("uci set firewall.%s=rule", sec),
		fmt.Sprintf("uci set firewall.%s.name='%s'", sec, name),
		fmt.Sprintf("uci set firewall.%s.src='%s'", sec, state.Data["src"]),
		fmt.Sprintf("uci set firewall.%s.dest='%s'", sec, state.Data["dest"]),
		fmt.Sprintf("uci set firewall.%s.dest_port='%s'", sec, state.Data["dest_port"]),
		fmt.Sprintf("uci set firewall.%s.target='%s'", sec, target),
		"uci commit firewall",
		"/etc/init.d/firewall reload",
	}

	SSHExec(strings.Join(cmds, " && "))
	session.GlobalStore.Delete(userID, "fw_wizard")

	return HandleFwMenu(c)
}
