package openclash

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yingxiaomo/homeops/pkg/utils"
	tele "gopkg.in/telebot.v3"
)

func HandleMenu(c tele.Context) error {
	if !utils.IsAdmin(c.Sender().ID) {
		return c.Send("⛔ 此功能仅限管理员使用。")
	}

	client := NewClient()

	cfg, err := client.GetConfig()
	statusTxt := "✅ 运行中"
	if err != nil {
		statusTxt = fmt.Sprintf("❌ 错误: %v", err)
	}

	mode := "?"
	if cfg != nil {
		if m, ok := cfg["mode"].(string); ok {
			mode = m
		}
	}

	txt := fmt.Sprintf("🚀 **OpenClash 面板**\n-------------------\n状态: %s\n模式: `%s`", statusTxt, utils.EscapeMarkdown(mode))

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("⚙️ 模式: "+mode, "clash_mode")),
		menu.Row(menu.Data("📊 状态", "clash_status"), menu.Data("🌍 节点", "clash_groups")),
		menu.Row(menu.Data("🧰 工具箱", "clash_tools")),
		menu.Row(menu.Data("🔙 返回主控台", "start_main")),
	)

	return c.EditOrSend(txt, menu, tele.ModeMarkdown)
}

func HandleCallback(c tele.Context, data string) error {
	switch {
	case data == "clash_main":
		return HandleMenu(c)
	case data == "clash_mode":
		return handleModeMenu(c)
	case data == "clash_status":
		return handleStatus(c)
	case data == "clash_groups":
		return handleGroups(c)
	case data == "clash_tools":
		return handleTools(c)
	case data == "clash_reload":
		return handleToolAction(c, "reload")
	case data == "clash_flush_fakeip":
		return handleToolAction(c, "fakeip")
	case data == "clash_flush_conns":
		return handleToolAction(c, "conns")
	case data == "wrt_ai_clash":
		return HandleAIAnalyze(c)
	case data == "clash_speedtest_all":
		return handleSpeedtestAll(c, "")
	case data == "clash_toggle_debug":
		return handleToggleDebug(c)
	case strings.HasPrefix(data, "clash_setm_"):
		return handleSetMode(c, data[11:])
	case strings.HasPrefix(data, "G_"):
		return handleListNodes(c, data[2:])
	case strings.HasPrefix(data, "S_"):
		parts := strings.Split(data[2:], "|")
		if len(parts) == 2 {
			return handleSetNode(c, parts[0], parts[1])
		}
	case strings.HasPrefix(data, "clash_testall_"):
		return handleSpeedtestAll(c, data[14:])
	}

	return c.Respond()
}

func handleModeMenu(c tele.Context) error {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("Rule (规则)", "clash_setm_rule"), menu.Data("Global (全局)", "clash_setm_global")),
		menu.Row(menu.Data("Direct (直连)", "clash_setm_direct"), menu.Data("Script (脚本)", "clash_setm_script")),
		menu.Row(menu.Data("🔙 返回", "clash_main")),
	)
	return c.Edit("🔄 **请选择运行模式**", menu, tele.ModeMarkdown)
}

func handleSetMode(c tele.Context, mode string) error {
	client := NewClient()
	if len(mode) > 0 {
		mode = strings.ToUpper(mode[:1]) + mode[1:]
	}

	err := client.PatchConfig(map[string]interface{}{
		"mode": mode,
	})

	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "切换失败: " + err.Error()})
	}

	c.Respond(&tele.CallbackResponse{Text: "已切换为 " + mode})
	return HandleMenu(c)
}

func fmtBytes(size float64) string {
	power := 1024.0
	n := 0
	powerLabels := []string{"", "K", "M", "G", "T"}
	for size > power {
		size /= power
		n++
	}
	if n >= len(powerLabels) {
		n = len(powerLabels) - 1
	}
	return fmt.Sprintf("%.2f %sB", size, powerLabels[n])
}

func handleStatus(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "获取状态中..."})
	client := NewClient()
	ver, err := client.GetVersion()
	conns, err2 := client.GetConnections()

	vStr := "Unknown"
	pStr := "Unknown"
	if err == nil {
		if v, ok := ver["version"].(string); ok {
			vStr = v
		}
		if p, ok := ver["premium"].(bool); ok {
			if p {
				pStr = "是"
			} else {
				pStr = "否"
			}
		}
	}

	connCount := 0
	uploadTotal := 0.0
	downloadTotal := 0.0

	if err2 == nil {
		if cList, ok := conns["connections"].([]interface{}); ok {
			connCount = len(cList)
			for _, item := range cList {
				if cmap, ok := item.(map[string]interface{}); ok {
					if u, ok := cmap["upload"].(float64); ok {
						uploadTotal += u
					}
					if d, ok := cmap["download"].(float64); ok {
						downloadTotal += d
					}
				}
			}
		}
	}

	txt := fmt.Sprintf("📊 **Clash 状态监控**\n-------------------\n🛠 版本: %s\n💎 Premium内核: %s\n-------------------\n🔗 当前活跃连接: %d\n🚀 实时上传: N/A\n⏬ 实时下载: N/A\n-------------------\n📦 当前会话总流量:\n   ⬆️ 上传: %s\n   ⬇️ 下载: %s",
		utils.EscapeMarkdown(vStr), pStr, connCount, fmtBytes(uploadTotal), fmtBytes(downloadTotal))

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("🔙 返回主菜单", "clash_main")),
	)
	return c.Edit(txt, menu, tele.ModeMarkdown)
}

func handleGroups(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "获取节点中..."})
	client := NewClient()
	proxies, err := client.GetProxies()
	if err != nil {
		return c.Edit("❌ 获取节点失败: " + err.Error())
	}

	txt := "请选择一个代理组 (Proxy Group):"

	menu := &tele.ReplyMarkup{}
	var rows []tele.Row
	var currentRow []tele.Btn

	excludeKeywords := []string{"Apple", "Microsoft", "Google", "Telegram", "Steam", "Speedtest", "Reject", "Direct", "Recycle", "Hijacking", "Video", "Media", "AD", "Bybit"}

	if pMap, ok := proxies["proxies"].(map[string]interface{}); ok {
		var names []string
		for name := range pMap {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			info := pMap[name].(map[string]interface{})
			if typeStr, ok := info["type"].(string); ok {
				if typeStr == "Selector" && name != "DIRECT" && name != "REJECT" && name != "GLOBAL" {
					skip := false
					for _, k := range excludeKeywords {
						if strings.Contains(strings.ToLower(name), strings.ToLower(k)) {
							skip = true
							break
						}
					}
					if !skip {
						currentRow = append(currentRow, menu.Data(name, "G_"+name))
						if len(currentRow) == 4 {
							rows = append(rows, menu.Row(currentRow...))
							currentRow = []tele.Btn{}
						}
					}
				}
			}
		}
	}

	if len(currentRow) > 0 {
		rows = append(rows, menu.Row(currentRow...))
	}
	rows = append(rows, menu.Row(menu.Data("🔙 返回", "clash_main")))
	menu.Inline(rows...)

	return c.Edit(txt, menu, tele.ModeMarkdown)
}

func handleListNodes(c tele.Context, groupName string) error {
	c.Respond(&tele.CallbackResponse{Text: "获取组信息..."})
	client := NewClient()
	proxies, err := client.GetProxies()
	if err != nil {
		return c.Edit("❌ API Error")
	}

	pMap, ok := proxies["proxies"].(map[string]interface{})
	if !ok {
		return c.Edit("❌ Data Error")
	}

	groupInfo, ok := pMap[groupName].(map[string]interface{})
	if !ok {
		return c.Edit("❌ Group Not Found")
	}

	allNodes, _ := groupInfo["all"].([]interface{})
	nowSelected, _ := groupInfo["now"].(string)

	menu := &tele.ReplyMarkup{}
	var rows []tele.Row
	var currentRow []tele.Btn

	for _, n := range allNodes {
		nodeName, ok := n.(string)
		if !ok {
			continue
		}

		// Get delay from history
		delay := 0
		if nodeInfo, ok := pMap[nodeName].(map[string]interface{}); ok {
			if hist, ok := nodeInfo["history"].([]interface{}); ok && len(hist) > 0 {
				if last, ok := hist[len(hist)-1].(map[string]interface{}); ok {
					if d, ok := last["delay"].(float64); ok {
						delay = int(d)
					}
				}
			}
		}

		delayStr := ""
		if delay > 0 {
			delayStr = fmt.Sprintf("(%dms)", delay)
		}

		label := nodeName + " " + delayStr
		if nodeName == nowSelected {
			label = "✅ " + label
		}

		data := fmt.Sprintf("S_%s|%s", groupName, nodeName)
		// Check length
		if len(data) > 64 {
		}

		currentRow = append(currentRow, menu.Data(label, data))
		if len(currentRow) == 2 {
			rows = append(rows, menu.Row(currentRow...))
			currentRow = []tele.Btn{}
		}
	}

	if len(currentRow) > 0 {
		rows = append(rows, menu.Row(currentRow...))
	}

	rows = append(rows, menu.Row(menu.Data("⚡ 一键测速所有节点", "clash_testall_"+groupName)))
	rows = append(rows, menu.Row(menu.Data("🔙 返回组列表", "clash_groups")))
	menu.Inline(rows...)

	return c.Edit(fmt.Sprintf("当前组: %s\n当前节点: %s\n请点击选择新节点:", utils.EscapeMarkdown(groupName), utils.EscapeMarkdown(nowSelected)), menu, tele.ModeMarkdown)
}

func handleSetNode(c tele.Context, group, node string) error {
	client := NewClient()
	err := client.PutProxy(group, node)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "切换失败: " + err.Error()})
	}
	c.Respond(&tele.CallbackResponse{Text: "Switched to " + node})
	return handleListNodes(c, group)
}

func handleTools(c tele.Context) error {
	client := NewClient()
	cfg, _ := client.GetConfig()
	logLevel := "unknown"
	if cfg != nil {
		if l, ok := cfg["log-level"].(string); ok {
			logLevel = l
		}
	}

	debugBtnText := fmt.Sprintf("🐛 切换调试模式 (当前: %s)", logLevel)

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("⚡ 全局节点测速", "clash_speedtest_all")),
		menu.Row(menu.Data("🤖 AI 分析内核日志", "clash_ai_analyze")),
		menu.Row(menu.Data(debugBtnText, "clash_toggle_debug")),
		menu.Row(menu.Data("♻️ 重载配置 (含清DNS)", "clash_reload")),
		menu.Row(menu.Data("✂️ 断开所有连接", "clash_flush_conns")),
		menu.Row(menu.Data("🧹 清除 FakeIP 缓存", "clash_flush_fakeip")),
		menu.Row(menu.Data("🔙 返回主菜单", "clash_main")),
	)
	return c.Edit("🛠 工具箱操作:", menu, tele.ModeMarkdown)
}

func handleToolAction(c tele.Context, action string) error {
	client := NewClient()
	var err error
	var msg string

	switch action {
	case "reload":
		c.Respond(&tele.CallbackResponse{Text: "正在重载配置..."})
		err = client.ReloadConfig()
		msg = "配置已重载"
	case "fakeip":
		c.Respond(&tele.CallbackResponse{Text: "正在清除 FakeIP..."})
		err = client.FlushFakeIP()
		msg = "FakeIP 缓存已清除"
	case "conns":
		c.Respond(&tele.CallbackResponse{Text: "正在断开连接..."})
		err = client.FlushConnections()
		msg = "所有连接已断开"
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("🔙 返回工具箱", "clash_tools")),
	)

	if err != nil {
		return c.Edit("❌ 操作失败: "+err.Error(), menu)
	}

	return c.Edit("✅ "+msg, menu)
}

func handleToggleDebug(c tele.Context) error {
	client := NewClient()
	cfg, err := client.GetConfig()
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "无法获取配置"})
	}

	currentLevel := "info"
	if l, ok := cfg["log-level"].(string); ok {
		currentLevel = l
	}

	newLevel := "debug"
	if currentLevel == "debug" {
		newLevel = "info"
	}

	err = client.PatchConfig(map[string]interface{}{"log-level": newLevel})
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "切换失败"})
	}

	c.Respond(&tele.CallbackResponse{Text: "日志级别已切换为: " + newLevel, ShowAlert: true})
	return handleTools(c)
}

func handleSpeedtestAll(c tele.Context, groupName string) error {
	c.Respond(&tele.CallbackResponse{Text: "正在测速，请稍候...", ShowAlert: true})
	client := NewClient()
	proxies, err := client.GetProxies()
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "获取节点失败"})
	}

	var targets []string

	if groupName != "" {
		// Test specific group
		if pMap, ok := proxies["proxies"].(map[string]interface{}); ok {
			if group, ok := pMap[groupName].(map[string]interface{}); ok {
				if all, ok := group["all"].([]interface{}); ok {
					for _, n := range all {
						if name, ok := n.(string); ok {
							targets = append(targets, name)
						}
					}
				}
			}
		}
	} else {
		if pMap, ok := proxies["proxies"].(map[string]interface{}); ok {
			for name, info := range pMap {
				if iMap, ok := info.(map[string]interface{}); ok {
					if _, hasAll := iMap["all"]; !hasAll {
						targets = append(targets, name)
					}
				}
			}
		}
	}

	batchSize := 10

	for i := 0; i < len(targets); i += batchSize {
		end := i + batchSize
		if end > len(targets) {
			end = len(targets)
		}

		var wg sync.WaitGroup
		for _, node := range targets[i:end] {
			wg.Add(1)
			go func(n string) {
				defer wg.Done()
				client.GetProxyDelay(n)
			}(node)
		}
		wg.Wait()
		time.Sleep(200 * time.Millisecond)
	}

	if groupName != "" {
		c.Respond(&tele.CallbackResponse{Text: "测速完成"})
		return handleListNodes(c, groupName)
	} else {
		c.Respond(&tele.CallbackResponse{Text: "全局测速完成！"})
		return handleTools(c)
	}
}
