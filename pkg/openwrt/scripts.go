package openwrt

import (
	"fmt"
	"path/filepath"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func HandleScriptsList(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{Text: "读取脚本列表..."})

	scriptDir := "/root/smart"
	res, _ := SSHExec(fmt.Sprintf("ls %s/*.sh 2>/dev/null", scriptDir))

	menu := &tele.ReplyMarkup{}
	if strings.TrimSpace(res) == "" {
		menu.Inline(menu.Row(menu.Data("🔙 返回", "wrt_main")))
		return c.Edit(fmt.Sprintf("目录 %s 下没有找到脚本。", scriptDir), menu)
	}

	var rows []tele.Row
	scripts := strings.Split(strings.TrimSpace(res), "\n")
	for _, s := range scripts {
		if s == "" {
			continue
		}
		name := filepath.Base(s)
		rows = append(rows, menu.Row(menu.Data(fmt.Sprintf("▶️ %s", name), "wrt_run_script", s)))
	}
	rows = append(rows, menu.Row(menu.Data("🔙 返回", "wrt_main")))
	menu.Inline(rows...)

	return c.Edit(fmt.Sprintf("📂 脚本列表 (%s):\n点击即可立即运行。", scriptDir), menu)
}

func HandleRunScript(c tele.Context) error {
	parts := strings.SplitN(c.Callback().Data, "|", 2)
	if len(parts) < 2 {
		return c.Respond(&tele.CallbackResponse{Text: "Error: Invalid script path"})
	}
	scriptPath := parts[1]

	c.Respond(&tele.CallbackResponse{Text: "正在运行脚本...", ShowAlert: true})
	c.Edit(fmt.Sprintf("⏳ 正在执行: %s\n请稍候...", scriptPath))

	res, _ := SSHExec(scriptPath)
	if len(res) > 3000 {
		res = res[:3000] + "\n... (输出过长已截断)"
	}

	resultText := fmt.Sprintf("✅ 执行完成: %s\n\n📝 输出:\n%s", scriptPath, res)
	if res == "" {
		resultText = fmt.Sprintf("✅ 执行完成 (无输出): %s", scriptPath)
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔙 返回脚本列表", "wrt_scripts_list")))
	return c.Edit(resultText, menu)
}
