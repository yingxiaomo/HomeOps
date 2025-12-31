package bot

import (
	"fmt"

	tele "gopkg.in/telebot.v3"
)

// Info Command
func (b *Bot) HandleInfo(c tele.Context) error {
	info := fmt.Sprintf(
		"ℹ️ **Bot Info**\n\n"+
			"ID: `%d`\n"+
			"Username: @%s\n"+
			"Go Version: 1.22\n"+
			"Framework: Telebot v3",
		b.TeleBot.Me.ID,
		b.TeleBot.Me.Username,
	)
	return c.Send(info, tele.ModeMarkdown)
}
