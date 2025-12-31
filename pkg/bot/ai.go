package bot

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/yingxiaomo/homeops/pkg/openwrt"
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
	return c.Send("🚪 **AI 模式已关闭**")
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

	resp, err := b.Gemini.GenerateContent(context.Background(), c.Text(), nil)
	if err != nil {
		_, err = b.TeleBot.Edit(msg, fmt.Sprintf("❌ Error: %v", err))
		return err
	}

	if len(resp) > 4000 {
		resp = resp[:4000] + "..."
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🚪 退出 AI 模式", "ai_toggle")))

	_, err = b.TeleBot.Edit(msg, resp, tele.ModeMarkdown, menu)
	if err != nil {
		b.TeleBot.Edit(msg, resp, menu)
	}
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

	if len(resp) > 4000 {
		resp = resp[:4000] + "..."
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🚪 退出 AI 模式", "ai_toggle")))

	_, err = b.TeleBot.Edit(msg, resp, tele.ModeMarkdown, menu)
	if err != nil {
		b.TeleBot.Edit(msg, resp, menu)
	}
	return nil
}
