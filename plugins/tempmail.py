import logging
import aiohttp
from telegram import Update, InlineKeyboardButton, InlineKeyboardMarkup
from telegram.ext import ContextTypes, CommandHandler, CallbackQueryHandler
from utils.permissions import is_admin

logger = logging.getLogger(__name__)

API_BASE = "https://www.1secmail.com/api/v1/"

user_mailboxes = {}

async def fetch_json(url):
    try:
        async with aiohttp.ClientSession() as session:
            async with session.get(url) as resp:
                if resp.status == 200:
                    return await resp.json()
    except Exception as e:
        logger.error(f"HTTP Request Error: {e}")
    return None

async def get_random_mailbox():
    data = await fetch_json(f"{API_BASE}?action=genRandomMailbox&count=1")
    return data[0] if data else None

async def get_messages(email):
    login, domain = email.split('@')
    data = await fetch_json(f"{API_BASE}?action=getMessages&login={login}&domain={domain}")
    return data if data is not None else []

async def get_message_content(email, msg_id):
    login, domain = email.split('@')
    data = await fetch_json(f"{API_BASE}?action=readMessage&login={login}&domain={domain}&id={msg_id}")
    return data

async def mail_menu(update: Update, context: ContextTypes.DEFAULT_TYPE):
    user = update.effective_user
    if not is_admin(user.id):
        await update.message.reply_text("⛔ 此功能仅限管理员使用。")
        return

    user_id = user.id
    current_mail = user_mailboxes.get(user_id)
    
    txt = "📧 **临时邮箱 (1secmail)**\n-------------------\n"
    
    keyboard = []
    
    if current_mail:
        txt += f"📫 当前邮箱: `{current_mail}`\n"
        keyboard.append([InlineKeyboardButton("🔄 刷新收件箱", callback_data="mail_refresh")])
  
        keyboard.append([InlineKeyboardButton("🆕 生成新邮箱", callback_data="mail_new")])
    else:
        txt += "尚未分配邮箱，请点击下方按钮生成。"
        keyboard.append([InlineKeyboardButton("🆕 生成新邮箱", callback_data="mail_new")])
        
    keyboard.append([InlineKeyboardButton("🔙 返回主控台", callback_data="start_main")])
    
    reply_markup = InlineKeyboardMarkup(keyboard)
    
    if update.callback_query:
        await update.callback_query.answer()
        try:
            await update.callback_query.edit_message_text(txt, reply_markup=reply_markup, parse_mode="Markdown")
        except:
            pass
    else:
        await update.message.reply_text(txt, reply_markup=reply_markup, parse_mode="Markdown")

async def mail_new(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer("正在生成...")
    
    new_mail = await get_random_mailbox()
    if new_mail:
        user_mailboxes[query.from_user.id] = new_mail
        await mail_menu(update, context)
    else:
        await query.edit_message_text("❌ 生成失败，请稍后再试。")

async def mail_refresh(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    user_id = query.from_user.id
    current_mail = user_mailboxes.get(user_id)
    
    if not current_mail:
        await mail_menu(update, context)
        return

    await query.answer("检查新邮件...")
    
    msgs = await get_messages(current_mail)
    
    keyboard = []
    txt = f"📧 **收件箱** ({current_mail})\n-------------------\n"
    
    if not msgs:
        txt += "📭 暂无新邮件。"
    else:
        for m in msgs[:5]: 
            subject = m.get('subject') or '(无主题)'
            sender = m.get('from')
            mid = m.get('id')
            date = m.get('date').split()[0] # simple date
            txt += f"📩 `{sender}`\n└ {subject}\n"
            keyboard.append([InlineKeyboardButton(f"👁️ 阅读: {subject[:10]}...", callback_data=f"mail_read_{mid}")])
    
    keyboard.append([InlineKeyboardButton("🔄 刷新", callback_data="mail_refresh")])
    keyboard.append([InlineKeyboardButton("🔙 返回", callback_data="mail_main")])
    
    await query.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(keyboard), parse_mode="Markdown")

async def mail_read(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    user_id = query.from_user.id
    current_mail = user_mailboxes.get(user_id)
    msg_id = query.data.replace("mail_read_", "")
    
    if not current_mail:
        await query.answer("Session expired.")
        await mail_menu(update, context)
        return

    await query.answer("加载内容...")
    
    msg_data = await get_message_content(current_mail, msg_id)
    if not msg_data:
        await query.edit_message_text("❌ 读取邮件失败。")
        return

    subject = msg_data.get('subject')
    sender = msg_data.get('from')
    date = msg_data.get('date')
    body = msg_data.get('textBody') or "无文本内容"
    
    txt = (
        f"📨 **邮件详情**\n"
        f"**From**: `{sender}`\n"
        f"**Date**: {date}\n"
        f"**Subject**: {subject}\n"
        f"-------------------\n"
        f"{body[:3000]}"
    )
    if len(body) > 3000: txt += "\n...(已截断)"
    
    keyboard = [[InlineKeyboardButton("🔙 返回收件箱", callback_data="mail_refresh")]]
    await query.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(keyboard), parse_mode="Markdown")

async def handle_callback(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    data = query.data
    
    if data == "mail_main":
        await mail_menu(update, context)
    elif data == "mail_new":
        await mail_new(update, context)
    elif data == "mail_refresh":
        await mail_refresh(update, context)
    elif data.startswith("mail_read_"):
        await mail_read(update, context)

handlers = [
    CommandHandler("mail", mail_menu),
    CallbackQueryHandler(handle_callback, pattern="^mail_")
]