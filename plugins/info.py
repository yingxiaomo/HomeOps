from telegram import Update
from telegram.constants import ParseMode
from telegram.ext import ContextTypes, CommandHandler

async def get_id(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """
    Returns the current User ID and Chat ID.
    """
    user_id = update.effective_user.id
    chat_id = update.effective_chat.id
    chat_type = update.effective_chat.type
    
    msg = (
        f"🆔 **ID 信息**\n\n"
        f"👤 **用户 ID**: `{user_id}`\n"
        f"💬 **会话 ID**: `{chat_id}`\n"
        f"📝 **类型**: {chat_type}"
    )
    
    if update.message and update.message.message_thread_id:
        msg += f"\n🧵 **话题 ID**: `{update.message.message_thread_id}`"

    await update.message.reply_text(msg, parse_mode=ParseMode.MARKDOWN)

handlers = [
    CommandHandler("id", get_id),
]

