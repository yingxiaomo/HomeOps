package openclash

import (
	"fmt"

	tele "gopkg.in/telebot.v3"
)

func HandleMenu(c tele.Context) error {
	// Need client instance. In a real app, inject it.
	// For this quick port, create new or use singleton.
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

	txt := fmt.Sprintf("🚀 **OpenClash 控制台**\n-------------------\n状态: %s\n模式: `%s`", statusTxt, mode)
	
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("🔄 切换模式", "clash_mode"), menu.Data("📜 日志分析", "clash_log")),
		menu.Row(menu.Data("🔙 返回主控台", "start_main")),
	)
	
	return c.EditOrSend(txt, menu, tele.ModeMarkdown)
}

func HandleCallback(c tele.Context, data string) error {
	switch data {
	case "clash_main":
		return HandleMenu(c)
	case "clash_mode":
		return handleModeMenu(c)
	case "clash_log":
		return c.Edit("🔍 **日志分析**\n正在采集内核日志进行 AI 诊断... (模拟中)")
	}
	
	if len(data) > 11 && data[:11] == "clash_setm_" {
		return handleSetMode(c, data[11:])
	}
	
	return c.Respond()
}

func handleModeMenu(c tele.Context) error {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("Global", "clash_setm_global"), menu.Data("Rule", "clash_setm_rule")),
		menu.Row(menu.Data("Direct", "clash_setm_direct"), menu.Data("Script", "clash_setm_script")),
		menu.Row(menu.Data("🔙 返回", "clash_main")),
	)
	return c.Edit("🔄 **请选择运行模式**", menu)
}

func handleSetMode(c tele.Context, mode string) error {
	client := NewClient()
	err := client.PatchConfig(map[string]interface{}{
		"mode": mode,
	})
	
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "切换失败: " + err.Error()})
	}
	
	c.Respond(&tele.CallbackResponse{Text: "已切换为 " + mode})
	return HandleMenu(c)
}
