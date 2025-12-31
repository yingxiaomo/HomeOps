package bot

import (
	"fmt"
	"strings"

	"github.com/yingxiaomo/homeops/pkg/utils"
	tele "gopkg.in/telebot.v3"
)

func (b *Bot) HandleGrant(c tele.Context) error {
	if !utils.IsAdmin(c.Sender().ID) {
		return nil
	}

	args := c.Args()
	if len(args) < 2 {
		return c.Send("用法: /grant <user_id> <feature>\n例如: /grant 12345678 ai")
	}

	targetID := args[0]
	feature := strings.ToLower(args[1])

	if utils.GrantPermission(targetID, feature) {
		utils.SavePermissions()
		return c.Send(fmt.Sprintf("✅ 已授权用户 `%s` 使用 `%s` 功能。", targetID, feature), tele.ModeMarkdown)
	}

	return c.Send(fmt.Sprintf("⚠️ 用户 `%s` 已拥有 `%s` 权限。", targetID, feature), tele.ModeMarkdown)
}

func (b *Bot) HandleRevoke(c tele.Context) error {
	if !utils.IsAdmin(c.Sender().ID) {
		return nil
	}

	args := c.Args()
	if len(args) < 2 {
		return c.Send("用法: /revoke <user_id> <feature>")
	}

	targetID := args[0]
	feature := strings.ToLower(args[1])

	if utils.RevokePermission(targetID, feature) {
		utils.SavePermissions()
		return c.Send(fmt.Sprintf("🚫 已撤销用户 `%s` 的 `%s` 权限。", targetID, feature), tele.ModeMarkdown)
	}

	return c.Send(fmt.Sprintf("⚠️ 用户 `%s` 没有 `%s` 权限。", targetID, feature), tele.ModeMarkdown)
}

func (b *Bot) HandleListUsers(c tele.Context) error {
	if !utils.IsAdmin(c.Sender().ID) {
		return nil
	}

	perms := utils.GetPermissions()
	if len(perms) == 0 {
		return c.Send("📂 当前没有已授权用户。")
	}

	msg := "👥 **已授权用户列表**\n-------------------\n"
	for uid, features := range perms {
		msg += fmt.Sprintf("👤 `%s`: %s\n", uid, strings.Join(features, ", "))
	}

	return c.Send(msg, tele.ModeMarkdown)
}
