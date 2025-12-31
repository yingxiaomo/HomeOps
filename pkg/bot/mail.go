package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/yingxiaomo/homeops/pkg/utils"

	tele "gopkg.in/telebot.v3"
)

const MailAPIBase = "https://www.1secmail.com/api/v1/"

type MailMessage struct {
	ID      int    `json:"id"`
	From    string `json:"from"`
	Subject string `json:"subject"`
	Date    string `json:"date"`
}

type MailContent struct {
	ID       int    `json:"id"`
	From     string `json:"from"`
	Subject  string `json:"subject"`
	Date     string `json:"date"`
	TextBody string `json:"textBody"`
	HTMLBody string `json:"htmlBody"`
}

var userMailboxes = make(map[int64]string)

func fetchJSON(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

func (b *Bot) HandleMailMenu(c tele.Context) error {
	if !utils.IsAdmin(c.Sender().ID) {
		return c.Send("⛔ 此功能仅限管理员使用。")
	}

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
		txt += "尚未分配邮箱，请点击下方按钮生成。"
	}

	return c.EditOrSend(txt, menu, tele.ModeMarkdown)
}

func (b *Bot) HandleMailCallback(c tele.Context, data string) error {
	if strings.HasPrefix(data, "mail_read_") {
		return b.HandleMailRead(c, data)
	}

	switch data {
	case "mail_main":
		return b.HandleMailMenu(c)
	case "mail_new":
		c.Respond(&tele.CallbackResponse{Text: "正在生成..."})
		var mails []string
		err := fetchJSON(MailAPIBase+"?action=genRandomMailbox&count=1", &mails)
		if err == nil && len(mails) > 0 {
			userMailboxes[c.Sender().ID] = mails[0]
			return b.HandleMailMenu(c)
		}
		return c.Respond(&tele.CallbackResponse{Text: "❌ 生成失败，请稍后再试。"})
	case "mail_refresh":
		return b.HandleMailRefresh(c)
	}
	return c.Respond()
}

func (b *Bot) HandleMailRefresh(c tele.Context) error {
	userID := c.Sender().ID
	currentMail := userMailboxes[userID]
	if currentMail == "" {
		return b.HandleMailMenu(c)
	}

	c.Respond(&tele.CallbackResponse{Text: "检查新邮件..."})

	parts := strings.Split(currentMail, "@")
	if len(parts) != 2 {
		return c.Send("邮箱格式错误")
	}
	login, domain := parts[0], parts[1]

	var msgs []MailMessage
	url := fmt.Sprintf("%s?action=getMessages&login=%s&domain=%s", MailAPIBase, login, domain)
	err := fetchJSON(url, &msgs)
	if err != nil {
		return c.Send("获取邮件失败")
	}

	txt := fmt.Sprintf("📧 **收件箱** (%s)\n-------------------\n", currentMail)
	menu := &tele.ReplyMarkup{}
	var rows []tele.Row

	if len(msgs) == 0 {
		txt += "📭 暂无新邮件。"
	} else {
		for _, m := range msgs {
			txt += fmt.Sprintf("📩 [%s] %s\n", m.Date, m.Subject)
			rows = append(rows, menu.Row(menu.Data(fmt.Sprintf("👀 查看: %s", m.Subject), fmt.Sprintf("mail_read_%d", m.ID))))
		}
	}

	rows = append(rows, menu.Row(menu.Data("🔙 返回", "mail_main")))
	menu.Inline(rows...)
	return c.Edit(txt, menu, tele.ModeMarkdown)
}

func (b *Bot) HandleMailRead(c tele.Context, data string) error {
	userID := c.Sender().ID
	currentMail := userMailboxes[userID]
	if currentMail == "" {
		return b.HandleMailMenu(c)
	}

	idStr := strings.TrimPrefix(data, "mail_read_")
	parts := strings.Split(currentMail, "@")
	login, domain := parts[0], parts[1]

	c.Respond(&tele.CallbackResponse{Text: "读取内容..."})

	var content MailContent
	url := fmt.Sprintf("%s?action=readMessage&login=%s&domain=%s&id=%s", MailAPIBase, login, domain, idStr)
	err := fetchJSON(url, &content)
	if err != nil {
		menu := &tele.ReplyMarkup{}
		menu.Inline(menu.Row(menu.Data("🔙 返回收件箱", "mail_refresh")))
		return c.Edit("❌ 读取邮件内容失败", menu)
	}

	txt := fmt.Sprintf("📩 **邮件详情**\n**From:** %s\n**Subject:** %s\n**Date:** %s\n\n%s",
		utils.EscapeMarkdown(content.From), utils.EscapeMarkdown(content.Subject), utils.EscapeMarkdown(content.Date), content.TextBody)

	if len(txt) > 4000 {
		txt = txt[:4000] + "\n...(截断)"
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔙 返回收件箱", "mail_refresh")))

	return c.Edit(txt, menu, tele.ModeMarkdown)
}
