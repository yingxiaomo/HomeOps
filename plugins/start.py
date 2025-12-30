import json
import os
from telegram import Update, InlineKeyboardButton, InlineKeyboardMarkup
from telegram.ext import ContextTypes, CommandHandler, CallbackQueryHandler

def load_config():
    """Loads configuration for this plugin."""
    config_path = os.path.join("config", "plugin_start.json")
    if os.path.exists(config_path):
        with open(config_path, "r", encoding="utf-8") as f:
            return json.load(f)
    return {"welcome_message": "欢迎使用控制中心"}

async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Shows the simplified main dashboard menu."""
    keyboard = [
        [InlineKeyboardButton("🧠 AI 助手", callback_data="ai_mode_start")],
        [InlineKeyboardButton("🚀 OpenClash", callback_data="clash_main"),
         InlineKeyboardButton("📟 OpenWrt", callback_data="wrt_main")],
        [InlineKeyboardButton("📧 临时邮箱", callback_data="mail_main"),
         InlineKeyboardButton("🖼️ 贴纸转换", callback_data="sticker_main")]
    ]
    reply_markup = InlineKeyboardMarkup(keyboard)
    
    msg = "🏠 **HomeOps 控制台**"
    
    if update.callback_query:
        await update.callback_query.answer()
        await update.callback_query.edit_message_text(msg.replace("**", ""), reply_markup=reply_markup)
    else:
        await update.message.reply_text(msg.replace("**", ""), reply_markup=reply_markup)

async def handle_callback(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    data = query.data
    
    if data == "start_main":
        await start(update, context)

handlers = [
    CommandHandler("start", start),
    CallbackQueryHandler(handle_callback, pattern="^start_")
]

