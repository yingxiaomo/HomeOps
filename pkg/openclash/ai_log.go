package openclash

import (
	"context"
	"fmt"
	"time"

	"github.com/yingxiaomo/homeops/pkg/ai"
	"github.com/yingxiaomo/homeops/pkg/openwrt"
	"github.com/yingxiaomo/homeops/pkg/utils"
	tele "gopkg.in/telebot.v3"
)

func HandleAIAnalyze(c tele.Context) error {
	// Check Admin Permission
	if !utils.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "⛔ 仅限管理员使用", ShowAlert: true})
	}

	c.Respond(&tele.CallbackResponse{Text: "🚀 启动 OpenClash 诊断..."})

	// Update UI to show progress
	err := c.Edit("🔍 正在初始化诊断环境...")
	if err != nil {
		return err
	}
	msg := c.Message()

	// Run analysis asynchronously
	go func() {
		client := NewClient()

		// 1. Get current config
		config, err := client.GetConfig()
		originalLevel := "info"
		if err == nil && config != nil {
			if l, ok := config["log-level"].(string); ok {
				originalLevel = l
			}
		}

		// 2. Switch to debug if needed
		if originalLevel != "debug" {
			c.Bot().Edit(msg, fmt.Sprintf("⚙️ 当前级别为 %s，正在临时切换至 debug...", originalLevel))
			client.PatchConfig(map[string]interface{}{"log-level": "debug"})
			time.Sleep(5 * time.Second)
		}

		// 3. Collect logs
		c.Bot().Edit(msg, "📡 正在全量采集多源日志...")
		diagCmd := "echo '--- [KERNEL LOG (DEBUG MODE)] ---'; tail -n 100 /tmp/openclash.log 2>/dev/null; " +
			"echo '--- [STARTUP/PLUGIN LOG] ---'; tail -n 100 /tmp/openclash_start.log 2>/dev/null; " +
			"echo '--- [SYSTEM SYSLOG] ---'; logread | grep -E -i 'clash|openclash' | tail -n 100; " +
			"echo '--- [NETWORK STATUS] ---'; ubus call network.interface.wan status | grep -E 'up|address|pending'"

		logs, err := openwrt.SSHExec(diagCmd)

		// 4. Restore config
		if originalLevel != "debug" {
			client.PatchConfig(map[string]interface{}{"log-level": originalLevel})
		}

		if err != nil || logs == "" {
			c.Bot().Edit(msg, fmt.Sprintf("❌ 采集失败: %v", err), &tele.ReplyMarkup{
				InlineKeyboard: [][]tele.InlineButton{{
					{Text: "🔙 返回", Data: "clash_main"},
				}},
			})
			return
		}

		// 5. AI Analyze
		c.Bot().Edit(msg, "🤖 正在利用 Gemini 3.0 Pro 进行多维度联合分析...")

		prompt := fmt.Sprintf(
			"你是 OpenClash 专家。用户平时使用的日志等级是 '%s'，但为了本次诊断，"+
				"我已临时将等级提升至 'debug' 并抓取了以下 4 个维度的聚合数据。请进行深度分析：\n\n"+
				"分析要求：\n"+
				"1. 检查 KERNEL 部分是否有节点握手失败、TLS 证书问题或 DNS 查询超时。\n"+
				"2. 检查 STARTUP 部分是否有配置文件生成失败、订阅下载错误或内核权限问题。\n"+
				"3. 检查 SYSTEM 部分是否有路由器内存不足 (OOM) 或网络接口重置的情况。\n"+
				"4. 综合判断当前的上网故障原因，并给出中文建议。\n\n"+
				"诊断聚合数据：\n%s", originalLevel, logs)

		aiClient := ai.NewGeminiClient()
		resp, err := aiClient.GenerateContent(context.Background(), prompt, nil)
		if err != nil {
			c.Bot().Edit(msg, fmt.Sprintf("❌ 分析失败: %v", err), &tele.ReplyMarkup{
				InlineKeyboard: [][]tele.InlineButton{{
					{Text: "🔙 返回", Data: "clash_main"},
				}},
			})
			return
		}

		if len(resp) > 3800 {
			resp = resp[:3800] + "\n...(内容过长已截断)"
		}

		resultText := fmt.Sprintf("📋 **AI OpenClash 综合诊断报告**\n-------------------\n%s", resp)

		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🔙 返回", "clash_main")))

		c.Bot().Edit(msg, resultText, tele.ModeMarkdown, menu)
	}()

	return nil
}
