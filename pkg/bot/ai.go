package bot

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	tele "gopkg.in/telebot.v3"
)

func (b *Bot) HandleAI(c tele.Context) error {
	userID := c.Sender().ID
	
	// Toggle mode
	// Note: In real app, use Store. But here for simplicity:
	current := b.Store.Get(userID, "ai_mode")
	if current == nil {
		b.Store.Set(userID, "ai_mode", true)
		return c.Send("🧠 **AI 模式已开启**\n发送文本或图片即可对话。")
	}
	
	// If exists, toggle off
	b.Store.Set(userID, "ai_mode", nil) // remove
	return c.Send("🚪 **AI 模式已关闭**")
}

func (b *Bot) HandleText(c tele.Context) error {
	userID := c.Sender().ID
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
	
	// Markdown safe send
	_, err = b.TeleBot.Edit(msg, resp, tele.ModeMarkdown)
	if err != nil {
		b.TeleBot.Edit(msg, resp)
	}
	return nil
}

func (b *Bot) HandlePhoto(c tele.Context) error {
	userID := c.Sender().ID
	if b.Store.Get(userID, "ai_mode") == nil {
		return nil
	}

	msg, _ := b.TeleBot.Send(c.Sender(), "🤔 接收图片中...")

	// Download photo
	photo := c.Message().Photo
	// telebot.File() returns io.ReadCloser, not *telebot.File
	// We need to use Download on the File object from the message, OR
	// Use b.TeleBot.Download(&photo.File, path) directly.
	
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

	_, err = b.TeleBot.Edit(msg, resp, tele.ModeMarkdown)
	if err != nil {
		b.TeleBot.Edit(msg, resp)
	}
	return nil
}
