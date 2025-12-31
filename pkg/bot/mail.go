package bot

import (
	"fmt"
	"go_bot/pkg/utils"
	"math/rand"
	"time"

	tele "gopkg.in/telebot.v3"
)

// Simple in-memory mail storage
var userMailboxes = make(map[int64]string)

func (b *Bot) HandleMailMenu(c tele.Context) error {
	userID := c.Sender().ID
	currentMail := userMailboxes[userID]

	menu := &tele.ReplyMarkup{}
	rows := []tele.Row{}
	
	if currentMail != "" {
		rows = append(rows, menu.Row(menu.Data("🔄 刷新收件箱", "mail_refresh")))
		rows = append(rows, menu.Row(menu.Data("🆕 生成新邮箱", "mail_new")))
	} else {
		rows = append(rows, menu.Row(menu.Data("🆕 生成新邮箱", "mail_new")))
	}
	rows = append(rows, menu.Row(menu.Data("🔙 返回主控台", "start_main")))
	menu.Inline(rows...)

	txt := "📧 **临时邮箱 (1secmail)**\n-------------------\n"
	if currentMail != "" {
		txt += fmt.Sprintf("📫 当前邮箱: `%s`", currentMail)
	} else {
		txt += "尚未分配邮箱。"
	}

	return c.EditOrSend(txt, menu, tele.ModeMarkdown)
}

func (b *Bot) HandleMailCallback(c tele.Context, data string) error {
	switch data {
	case "mail_main":
		return b.HandleMailMenu(c)
	case "mail_new":
		// Mock generation
		rand.Seed(time.Now().UnixNano())
		name := utils.RandomString(8)
		userMailboxes[c.Sender().ID] = fmt.Sprintf("%s@1secmail.com", name)
		return b.HandleMailMenu(c)
	case "mail_refresh":
		c.Respond(&tele.CallbackResponse{Text: "暂无新邮件 (Mock)"})
		return nil
	}
	return c.Respond()
}
