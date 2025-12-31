package openwrt

import (
	"context"
	"fmt"
	"strings"

	"github.com/yingxiaomo/homeops/pkg/ai"
	"github.com/yingxiaomo/homeops/pkg/utils"
	tele "gopkg.in/telebot.v3"
)

// HandleAIAnalyze performs AI analysis on OpenWrt logs
func HandleAIAnalyze(c tele.Context) error {
	// Check Admin Permission
	if !utils.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "⛔ 仅限管理员使用", ShowAlert: true})
	}

	c.Respond(&tele.CallbackResponse{Text: "🚀 启动日志分析..."})

	// Update UI to show progress
	err := c.Edit("🔍 正在采集 OpenWrt 系统日志 (最后 100 行)...")
	if err != nil {
		return err
	}
	msg := c.Message()

	// Run analysis asynchronously
	go func() {
		// 1. Fetch Logs via SSH
		cmd := "logread | tail -n 100"
		logs, err := SSHExec(cmd)
		if err != nil {
			c.Bot().Edit(msg, fmt.Sprintf("❌ 采集失败: %v", err), &tele.ReplyMarkup{
				InlineKeyboard: [][]tele.InlineButton{{
					{Text: "🔙 返回", Data: "wrt_main"},
				}},
			})
			return
		}

		if strings.TrimSpace(logs) == "" {
			c.Bot().Edit(msg, "❌ 日志为空", &tele.ReplyMarkup{
				InlineKeyboard: [][]tele.InlineButton{{
					{Text: "🔙 返回", Data: "wrt_main"},
				}},
			})
			return
		}

		c.Bot().Edit(msg, "🤖 正在利用 Gemini 3.0 Pro 进行智能分析...")

		client := ai.NewGeminiClient()
		prompt := fmt.Sprintf("你是 OpenWrt 专家。请分析以下 OpenWrt 系统日志，指出潜在问题（如网络错误、系统异常、攻击尝试等），并给出中文建议：\n\n%s", logs)

		ctx := context.Background()
		resp, err := client.GenerateContent(ctx, prompt, nil)
		if err != nil {
			c.Bot().Edit(msg, fmt.Sprintf("❌ 分析失败: %v", err), &tele.ReplyMarkup{
				InlineKeyboard: [][]tele.InlineButton{{
					{Text: "🔙 返回", Data: "wrt_main"},
				}},
			})
			return
		}

		// 4. Display Results
		// Truncate if too long (Telegram limit is 4096 chars)
		if len(resp) > 3800 {
			resp = resp[:3800] + "\n...(内容过长已截断)"
		}

		resultText := fmt.Sprintf("📋 **AI OpenWrt 综合诊断报告**\n-------------------\n%s", resp)

		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🔙 返回", "wrt_main")))

		// Use ModeMarkdown for formatting
		c.Bot().Edit(msg, resultText, tele.ModeMarkdown, menu)
	}()

	return nil
}
