package openwrt

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yingxiaomo/homeops/pkg/ai"
	"github.com/yingxiaomo/homeops/pkg/utils"
	tele "gopkg.in/telebot.v3"
)

var (
	isAnalyzing bool
	analyzeLock sync.Mutex
)

func HandleAIAnalyze(c tele.Context) error {
	if !utils.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "⛔ 仅限管理员使用", ShowAlert: true})
	}

	analyzeLock.Lock()
	if isAnalyzing {
		analyzeLock.Unlock()
		return c.Respond(&tele.CallbackResponse{Text: "⏳ 正在进行中，请稍候...", ShowAlert: true})
	}
	isAnalyzing = true
	analyzeLock.Unlock()

	c.Respond(&tele.CallbackResponse{Text: "🚀 启动日志分析..."})

	err := c.Edit("🔍 正在采集 OpenWrt 系统日志 (最后 100 行)...")
	if err != nil {
		analyzeLock.Lock()
		isAnalyzing = false
		analyzeLock.Unlock()
		return err
	}
	msg := c.Message()

	go func() {
		defer func() {
			analyzeLock.Lock()
			isAnalyzing = false
			analyzeLock.Unlock()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

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

		resp, err := client.GenerateContent(ctx, prompt, nil)
		if err != nil {
			errMsg := fmt.Sprintf("❌ 分析失败: %v", err)
			if ctx.Err() == context.DeadlineExceeded {
				errMsg = "❌ 分析超时，请稍后重试"
			}
			c.Bot().Edit(msg, errMsg, &tele.ReplyMarkup{
				InlineKeyboard: [][]tele.InlineButton{{
					{Text: "🔙 返回", Data: "wrt_main"},
				}},
			})
			return
		}

		resultText := fmt.Sprintf("📋 **AI OpenWrt 综合诊断报告**\n-------------------\n%s", resp)

		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🔙 返回", "wrt_main")))

		utils.SendLongMessage(c, msg, resultText, menu)
	}()

	return nil
}
