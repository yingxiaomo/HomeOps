package bot

import (
	"fmt"
	"os"

	"image/png"
	_ "image/png"

	"image"

	_ "golang.org/x/image/webp"

	tele "gopkg.in/telebot.v3"
)

func (b *Bot) HandleStickerMenu(c tele.Context) error {
	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔙 返回主控台", "start_main")))

	txt := "🖼️ **贴纸转换工具**\n-------------------\n✨ **使用方法**：\n直接发送 **静态贴纸**，机器人会自动转换为 PNG。"
	return c.EditOrSend(txt, menu, tele.ModeMarkdown)
}

func (b *Bot) HandleStickerCallback(c tele.Context, data string) error {
	if data == "sticker_main" {
		return b.HandleStickerMenu(c)
	}
	return c.Respond()
}

func (b *Bot) HandleSticker(c tele.Context) error {
	if c.Message().Sticker.Animated || c.Message().Sticker.Video {
		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🔙 返回", "sticker_main")))
		return c.Send("❌ 仅支持静态贴纸。", menu)
	}

	msg, _ := b.TeleBot.Send(c.Sender(), "⏳ 正在转换...")

	tmpInput := fmt.Sprintf("temp_%s.webp", c.Message().Sticker.UniqueID)
	defer os.Remove(tmpInput)

	if err := b.TeleBot.Download(&c.Message().Sticker.File, tmpInput); err != nil {
		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🔙 返回", "sticker_main")))
		_, _ = b.TeleBot.Edit(msg, "❌ 下载失败", menu)
		return err
	}

	f, err := os.Open(tmpInput)
	if err != nil {
		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🔙 返回", "sticker_main")))
		_, _ = b.TeleBot.Edit(msg, "❌ 读取文件失败", menu)
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🔙 返回", "sticker_main")))
		_, _ = b.TeleBot.Edit(msg, fmt.Sprintf("❌ 解码失败: %v", err), menu)
		return err
	}

	tmpOutput := fmt.Sprintf("sticker_%s.png", c.Message().Sticker.UniqueID)
	defer os.Remove(tmpOutput)

	out, err := os.Create(tmpOutput)
	if err != nil {
		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🔙 返回", "sticker_main")))
		_, _ = b.TeleBot.Edit(msg, "❌ 创建输出文件失败", menu)
		return err
	}

	if err := png.Encode(out, img); err != nil {
		out.Close()
		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🔙 返回", "sticker_main")))
		_, _ = b.TeleBot.Edit(msg, "❌ 编码 PNG 失败", menu)
		return err
	}
	out.Close()

	doc := &tele.Document{
		File:     tele.FromDisk(tmpOutput),
		Caption:  "✅ 转换成功！",
		FileName: fmt.Sprintf("sticker_%s.png", c.Message().Sticker.UniqueID),
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔙 返回", "sticker_main")))

	_, err = b.TeleBot.Send(c.Sender(), doc, menu)
	b.TeleBot.Delete(msg)
	return err
}
