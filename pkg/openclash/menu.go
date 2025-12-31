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
	case "clash_log":
		return c.Edit("🔍 **日志分析**\n正在采集内核日志进行 AI 诊断... (模拟中)")
	}
	return c.Respond()
}
