package bot

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

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

	// Check if we have a log context to determine which menu to return to
	var menu *tele.ReplyMarkup
	logContext := b.Store.Get(userID, "ai_log_context")
	if logContext != nil {
		// For log analysis context, return to global main menu
		// This provides a cleaner exit experience
		menu = b.getMainMenu()
	} else {
		menu = b.getMainMenu()
	}

	// Clear log context after determining the menu
	b.Store.Delete(userID, "ai_log_context")

	txt := "🚪 **AI 模式已关闭**\n🤖 **HomeOps 已连接**\n\n请选择功能菜单："

	// 尝试直接编辑消息返回主菜单，实现无缝退出
	err := c.Edit(txt, menu, tele.ModeMarkdown)
	if err != nil {
		return c.Send(txt, menu, tele.ModeMarkdown)
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

	// Check if in batch input mode
	if b.Store.Get(userID, "batch_mode") != nil {
		return b.handleBatchMessage(c)
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

func (b *Bot) handleBatchMessage(c tele.Context) error {
	userID := c.Sender().ID

	// Get current messages
	messages := b.Store.Get(userID, "batch_messages")
	if messages == nil {
		messages = []string{}
	}

	msgs, ok := messages.([]string)
	if !ok {
		msgs = []string{}
	}

	// Add new message
	msgs = append(msgs, c.Text())
	b.Store.Set(userID, "batch_messages", msgs)

	// Send confirmation
	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("✅ 完成输入", "batch_end"), menu.Data("❌ 取消", "ai_toggle")))

	message := fmt.Sprintf("📝 已收集 %d 条消息\n\n最新消息: %s\n\n继续发送更多消息，或点击\"✅ 完成输入\"开始处理。", len(msgs), c.Text())
	return c.Send(message, menu)
}

func (b *Bot) HandleBatchStart(c tele.Context) error {
	userID := c.Sender().ID

	// Set batch input mode
	b.Store.Set(userID, "batch_mode", true)
	b.Store.Set(userID, "batch_messages", []string{})

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("✅ 完成输入", "batch_end"), menu.Data("❌ 取消", "ai_toggle")))

	return c.Edit("📝 **批量输入模式已开启**\n\n请发送多条消息，我会将它们收集起来一起处理。\n\n发送完成后点击\"✅ 完成输入\"按钮。", menu)
}

func (b *Bot) HandleBatchEnd(c tele.Context) error {
	userID := c.Sender().ID

	// Get collected messages
	messages := b.Store.Get(userID, "batch_messages")
	if messages == nil {
		return c.Edit("❌ 没有收集到任何消息")
	}

	msgs, ok := messages.([]string)
	if !ok || len(msgs) == 0 {
		return c.Edit("❌ 没有收集到任何消息")
	}

	// Clear batch mode
	b.Store.Set(userID, "batch_mode", nil)
	b.Store.Set(userID, "batch_messages", nil)

	// Combine all messages
	combinedText := strings.Join(msgs, "\n\n")

	// Enable AI mode for processing
	b.Store.Set(userID, "ai_mode", true)

	// Process the combined text as if it was a single message
	msg, _ := b.TeleBot.Send(c.Sender(), "🤔 正在处理批量输入...")

	// Build prompt with history if available
	prompt := combinedText
	history := ""
	if h := b.Store.Get(userID, "ai_history"); h != nil {
		if hStr, ok := h.(string); ok {
			history = hStr
			if len(history) > 20000 {
				history = history[len(history)-20000:]
			}
			prompt = history + "\nUser: " + combinedText
		}
	}

	// Check for log context
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
				freshLogs, logErr = openclash.GetDiagnosticLogs(false)
			}
			if logErr != nil {
				c.Send(fmt.Sprintf("⚠️ 无法获取最新日志: %v\n将基于历史进行回答。", logErr))
			} else {
				freshLogs = strings.ToValidUTF8(freshLogs, "�")
			}
			b.TeleBot.Edit(msg, "🤔 正在处理批量输入...")
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
			newHistory = "User: " + combinedText + "\n"
		} else {
			newHistory += "User: " + combinedText + "\n"
		}
		newHistory += "Model: " + resp + "\n"
		b.Store.Set(userID, "ai_history", newHistory)
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🚪 退出 AI 模式", "ai_toggle")))

	utils.SendLongMessage(c, msg, resp, menu)
	return nil
}
