import logging
from telegram import Update
from telegram.ext import ContextTypes, CommandHandler
from utils.permissions import grant_permission, revoke_permission, load_permissions, is_admin

logger = logging.getLogger(__name__)

async def grant_cmd(update: Update, context: ContextTypes.DEFAULT_TYPE):
    user = update.effective_user
    if not is_admin(user.id):
        return 

    args = context.args
    if len(args) < 2:
        await update.message.reply_text("用法: /grant <user_id> <feature>\n例如: /grant 12345678 ai")
        return

    target_id = args[0]
    feature = args[1].lower()

    if grant_permission(target_id, feature):
        await update.message.reply_text(f"✅ 已授权用户 `{target_id}` 使用 `{feature}` 功能。", parse_mode="Markdown")
    else:
        await update.message.reply_text(f"⚠️ 用户 `{target_id}` 已拥有 `{feature}` 权限。", parse_mode="Markdown")

async def revoke_cmd(update: Update, context: ContextTypes.DEFAULT_TYPE):
    user = update.effective_user
    if not is_admin(user.id):
        return

    args = context.args
    if len(args) < 2:
        await update.message.reply_text("用法: /revoke <user_id> <feature>")
        return

    target_id = args[0]
    feature = args[1].lower()

    if revoke_permission(target_id, feature):
        await update.message.reply_text(f"🚫 已撤销用户 `{target_id}` 的 `{feature}` 权限。", parse_mode="Markdown")
    else:
        await update.message.reply_text(f"⚠️ 用户 `{target_id}` 没有 `{feature}` 权限。", parse_mode="Markdown")

async def list_users_cmd(update: Update, context: ContextTypes.DEFAULT_TYPE):
    user = update.effective_user
    if not is_admin(user.id):
        return

    perms = load_permissions()
    if not perms:
        await update.message.reply_text("📂 当前没有已授权用户。" )
        return

    msg = "👥 **已授权用户列表**\n-------------------\n"
    for uid, features in perms.items():
        msg += f"👤 `{uid}`: {', '.join(features)}\n"
    
    await update.message.reply_text(msg, parse_mode="Markdown")

handlers = [
    CommandHandler("grant", grant_cmd),
    CommandHandler("revoke", revoke_cmd),
    CommandHandler("users", list_users_cmd)
]
