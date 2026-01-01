package openclash

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yingxiaomo/homeops/pkg/ai"
	"github.com/yingxiaomo/homeops/pkg/session"
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

	defer func() {
	}()

	c.Respond(&tele.CallbackResponse{Text: "🚀 启动 OpenClash 诊断..."})

	err := c.Edit("🔍 正在初始化诊断环境...")
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

		// For the first time, collect logs with debug level enabled.
		// For follow-ups, we can just grab the standard logs.
		logs, err := GetDiagnosticLogs(true)
		if err != nil {
			c.Bot().Edit(msg, fmt.Sprintf("❌ 采集失败: %v", err), &tele.ReplyMarkup{
				InlineKeyboard: [][]tele.InlineButton{{
					{Text: "🔙 返回", Data: "clash_main"},
				}},
			})
			return
		}

		c.Bot().Edit(msg, "🤖 正在利用 Gemini 3.0 Pro 进行多维度联合分析...")

		prompt := fmt.Sprintf(
			"你是 OpenClash 专家。我已临时开启 debug 模式并为你抓取了以下聚合数据。请进行深度分析：\n\n"+
				"分析要求：\n"+
				"1. 检查 KERNEL 部分是否有节点握手失败、TLS 证书问题或 DNS 查询超时。\n"+
				"2. 检查 STARTUP 部分是否有配置文件生成失败、订阅下载错误或内核权限问题。\n"+
				"3. 检查 SYSTEM 部分是否有路由器内存不足 (OOM) 或网络接口重置的情况。\n"+
				"4. 综合判断当前的上网故障原因，并给出中文建议。\n\n"+
				"诊断聚合数据：\n%s", logs)

		aiClient := ai.NewGeminiClient()
		resp, err := aiClient.GenerateContent(ctx, prompt, nil)
		if err != nil {
			errMsg := fmt.Sprintf("❌ 分析失败: %v", err)
			if ctx.Err() == context.DeadlineExceeded {
				errMsg = "❌ 分析超时，请稍后重试"
			}
			c.Bot().Edit(msg, errMsg, &tele.ReplyMarkup{
				InlineKeyboard: [][]tele.InlineButton{{
					{Text: "🔙 返回", Data: "clash_main"},
				}},
			})
			return
		}

		resultText := fmt.Sprintf("📋 **AI OpenClash 综合诊断报告**\n-------------------\n%s\n\n💡 **现在你可以直接发送消息继续咨询此问题。**", resp)

		// Enable AI mode and save history
		userID := c.Sender().ID
		session.GlobalStore.Set(userID, "ai_mode", true)
		session.GlobalStore.Set(userID, "ai_log_context", "openclash") // Set context for follow-ups
		
		// Save context for continuous chat
		history := fmt.Sprintf("User: %s\nModel: %s\n", prompt, resp)
		session.GlobalStore.Set(userID, "ai_history", history)

		menu := &tele.ReplyMarkup{}
		menu.Inline(
			menu.Row(menu.Data("🚪 退出 AI 模式", "ai_toggle")),
			menu.Row(menu.Data("🔙 返回", "clash_main")),
		)

		// Use utils.SendLongMessage to handle splitting and markdown safety
		utils.SendLongMessage(c, msg, resultText, menu)
	}()

	return nil
}
