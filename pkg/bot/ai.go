package bot

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/yingxiaomo/homeops/pkg/openclash"
	"github.com/yingxiaomo/homeops/pkg/openwrt"
	"github.com/yingxiaomo/homeops/pkg/utils"
	tele "gopkg.in/telebot.v3"
)

func (b *Bot) HandleAI(c tele.Context) error {
	userID := c.Sender().ID

	current := b.Store.Get(userID, "ai_mode")
	if current == nil {
		b.Store.Set(userID, "ai_mode", true)
		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🚪 退出 AI 模式", "ai_toggle")))
		return c.Send("🧠 **AI 模式已开启**\n发送文本或图片即可对话。", menu)
	}

	b.Store.Set(userID, "ai_mode", nil)
	b.Store.Delete(userID, "ai_history")
	b.Store.Delete(userID, "ai_log_context") // Clear log context on exit

	menu := b.getMainMenu()

	hour := time.Now().Hour()
	var timeGreeting string
	switch {
	case hour >= 0 && hour < 5:
		timeGreeting = "深夜了，注意休息 🌙"
	case hour >= 5 && hour < 9:
		timeGreeting = "早上好，新的一天加油 ☀️"
	case hour >= 9 && hour < 12:
		timeGreeting = "上午好 ☕"
	case hour >= 12 && hour < 14:
		timeGreeting = "中午好，记得按时吃饭 🍱"
	case hour >= 14 && hour < 18:
		timeGreeting = "下午好，喝杯茶提提神吧 🍵"
	case hour >= 18 && hour < 23:
		timeGreeting = "晚上好，辛苦一天了 🌃"
	default:
		timeGreeting = "你好 👋"
	}
	txt := fmt.Sprintf("🚪 **AI 模式已关闭**\n🤖 **HomeOps 已连接**\n\n%s\n\n请选择功能菜单：", timeGreeting)

	// 尝试直接编辑消息返回主菜单，实现无缝退出
	err := c.Edit(txt, menu)
	if err != nil {
		return c.Send(txt, menu)
	}
	return nil
}

func (b *Bot) HandleText(c tele.Context) error {
	userID := c.Sender().ID

	if state := b.Store.Get(userID, "wrt_net_state"); state != nil {
		if s, ok := state.(string); ok {
			return openwrt.HandleNetInput(c, s)
		}
	}

	if state := b.Store.Get(userID, "fw_wizard"); state != nil {
		return openwrt.HandleFwWizardInput(c, c.Text())
	}

	if state := b.Store.Get(userID, "adg_wizard"); state != nil {
		if s, ok := state.(map[string]interface{}); ok {
			if openwrt.HandleAdgWizardInput(c, s) {
				return nil
			}
		}
	}

	if b.Store.Get(userID, "ai_mode") == nil {
		return nil
	}

	msg, _ := b.TeleBot.Send(c.Sender(), "🤔 思考中...")

	// --- Dynamic Log Fetching based on Context ---
	freshLogs := ""
	var logErr error
	logContext := ""
	if ctx := b.Store.Get(userID, "ai_log_context"); ctx != nil {
		if s, ok := ctx.(string); ok {
			logContext = s
			b.TeleBot.Edit(msg, fmt.Sprintf("🔄 正在刷新 %s 最新日志...", logContext))
			switch logContext {
			case "openwrt":
				freshLogs, logErr = openwrt.GetLogs(100)
			case "openclash":
				// For follow-ups, don't force debug level to avoid repeated switching.
				freshLogs, logErr = openclash.GetDiagnosticLogs(false)
			}
			if logErr != nil {
				c.Send(fmt.Sprintf("⚠️ 无法获取最新日志: %v\n将基于历史进行回答。", logErr))
			} else {
				// Sanitize the logs to ensure they are valid UTF-8
				freshLogs = strings.ToValidUTF8(freshLogs, "�")
			}
			b.TeleBot.Edit(msg, "🤔 思考中...")
		}
	}
	// --- End of Dynamic Log Fetching ---

	// Build prompt with history if available
	prompt := c.Text()
	history := ""
	if h := b.Store.Get(userID, "ai_history"); h != nil {
		if hStr, ok := h.(string); ok {
			history = hStr
			// Limit history length to avoid token limits (simple char limit for now)
			if len(history) > 20000 {
				history = history[len(history)-20000:]
			}
			prompt = history + "\nUser: " + c.Text()
		}
	}

	if freshLogs != "" {
		prompt += fmt.Sprintf("\n\n--- [最新日志参考] ---\n%s\n--- [日志结束] ---", freshLogs)
	}

	resp, err := b.Gemini.GenerateContent(context.Background(), prompt, nil)
	if err != nil {
		_, err = b.TeleBot.Edit(msg, fmt.Sprintf("❌ Error: %v", err))
		return err
	}

	// Update history
	if history != "" || b.Store.Get(userID, "ai_mode") != nil {
		newHistory := history 
		if newHistory == "" {
			newHistory = "User: " + c.Text() + "\n"
		} else {
			newHistory += "User: " + c.Text() + "\n"
		}
		newHistory += "Model: " + resp + "\n"
		b.Store.Set(userID, "ai_history", newHistory)
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🚪 退出 AI 模式", "ai_toggle")))

	utils.SendLongMessage(c, msg, resp, menu)
	return nil
}

func (b *Bot) HandlePhoto(c tele.Context) error {
	userID := c.Sender().ID
	if b.Store.Get(userID, "ai_mode") == nil {
		return nil
	}

	msg, _ := b.TeleBot.Send(c.Sender(), "🤔 接收图片中...")

	photo := c.Message().Photo

	tmpFile := fmt.Sprintf("temp_ai_%d.jpg", userID)
	defer os.Remove(tmpFile)

	if err := b.TeleBot.Download(&photo.File, tmpFile); err != nil {
		_, err = b.TeleBot.Edit(msg, "❌ 下载图片失败")
		return err
	}

	imgBytes, err := ioutil.ReadFile(tmpFile)
	if err != nil {
		_, err = b.TeleBot.Edit(msg, "❌ 读取图片失败")
		return err
	}

	b.TeleBot.Edit(msg, "🤔 正在分析图片...")

	prompt := c.Message().Caption
	if prompt == "" {
		prompt = "Describe this image"
	}

	resp, err := b.Gemini.GenerateContent(context.Background(), prompt, imgBytes)
	if err != nil {
		_, err = b.TeleBot.Edit(msg, fmt.Sprintf("❌ Error: %v", err))
		return err
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🚪 退出 AI 模式", "ai_toggle")))

	utils.SendLongMessage(c, msg, resp, menu)
	return nil
}
