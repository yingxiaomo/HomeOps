from telegram import InlineKeyboardButton, InlineKeyboardMarkup
from telegram.ext import ContextTypes
from utils.permissions import is_admin
from .helpers import safe_callback_answer
from .connection import _close_ssh

async def wrt_menu(update, context: ContextTypes.DEFAULT_TYPE):
    user = update.effective_user
    if not is_admin(user.id):
        await update.message.reply_text("⛔ 此功能仅限管理员使用。")
        return
    keyboard = [
        [InlineKeyboardButton("📈 系统状态", callback_data="wrt_status"),
         InlineKeyboardButton("🏠 当前 IP", callback_data="wrt_show_current_ips")],
        [InlineKeyboardButton("📱 联网设备", callback_data="wrt_devices"),
         InlineKeyboardButton("🌐 网络测试", callback_data="wrt_net_menu")],
        [InlineKeyboardButton("📜 运行脚本", callback_data="wrt_scripts_list"),
         InlineKeyboardButton("🔥 防火墙", callback_data="wrt_fw_menu")],
        [InlineKeyboardButton("🛡️ ADG 控制", callback_data="wrt_adg_menu"),
         InlineKeyboardButton("🔄 重启系统", callback_data="wrt_reboot_confirm")],
        [InlineKeyboardButton("🤖 AI 分析日志", callback_data="wrt_ai_analyze"),
         InlineKeyboardButton("🔙 返回", callback_data="wrt_exit")],
    ]
    reply_markup = InlineKeyboardMarkup(keyboard)
    msg = "📟 **OpenWrt 面板**"
    if update.callback_query:
        await safe_callback_answer(update.callback_query)
        await update.callback_query.edit_message_text(msg.replace("**", ""), reply_markup=reply_markup)
    else:
        await update.message.reply_text(msg.replace("**", ""), reply_markup=reply_markup)

async def wrt_exit(update, context: ContextTypes.DEFAULT_TYPE):
    _close_ssh()
    k = [
        [InlineKeyboardButton("🧠 AI 助手", callback_data="ai_mode_start")],
        [InlineKeyboardButton("🚀 OpenClash", callback_data="clash_main"),
         InlineKeyboardButton("📟 OpenWrt", callback_data="wrt_main")],
        [InlineKeyboardButton("📧 临时邮箱", callback_data="mail_main"),
         InlineKeyboardButton("🖼️ 贴纸转换", callback_data="sticker_main")]
    ]
    await update.callback_query.edit_message_text("🏠 HomeOps 控制台", reply_markup=InlineKeyboardMarkup(k))
