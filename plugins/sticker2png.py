import io
import logging
from PIL import Image
from telegram import Update, InlineKeyboardButton, InlineKeyboardMarkup
from telegram.ext import ContextTypes, MessageHandler, filters, CommandHandler, CallbackQueryHandler

logger = logging.getLogger(__name__)

async def sticker_to_png(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Converts a static sticker to PNG."""
    sticker = update.message.sticker
    
    if sticker.is_animated or sticker.is_video:
        await update.message.reply_text("❌ 抱歉，目前仅支持静态贴纸转换。" )
        return

    status_msg = await update.message.reply_text("⏳ 正在处理...")

    try:
        new_file = await context.bot.get_file(sticker.file_id)
        
        f = io.BytesIO()
        await new_file.download_to_memory(f)
        f.seek(0)
        
        try:
            img = Image.open(f)
            png_io = io.BytesIO()
            img.save(png_io, 'PNG')
            png_io.seek(0)
            
            original_name = f"sticker_{sticker.file_unique_id}.png"
            await update.message.reply_document(
                document=png_io,
                filename=original_name,
                caption="✅ 转换成功！",
                quote=True
            )
            await status_msg.delete()
            
        except Exception as e:
            logger.error(f"Image conversion error: {e}")
            await status_msg.edit_text("❌ 图片转换失败，格式可能不受支持。" )

    except Exception as e:
        logger.error(f"Download error: {e}")
        await status_msg.edit_text("❌ 下载贴纸失败。" )

async def sticker_menu(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Shows instructions for sticker conversion."""
    query = update.callback_query
    txt = (
        "🖼️ **贴纸转换工具**\n"
        "-------------------\n"
        "✨ **使用方法**：\n"
        "直接在私聊中发送任何 **静态贴纸** 给机器人，它将自动为您转换成透明背景的 PNG 文件并回复。\n\n"
        "⚠️ 目前仅支持静态 WebP 贴纸。"
    )
    keyboard = [[InlineKeyboardButton("🔙 返回主控台", callback_data="start_main")]]
    
    if query:
        await query.answer()
        await query.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(keyboard), parse_mode="Markdown")
    else:
        await update.message.reply_text(txt, reply_markup=InlineKeyboardMarkup(keyboard), parse_mode="Markdown")

async def handle_callback(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    if query.data == "sticker_main":
        await sticker_menu(update, context)

# Register handlers
handlers = [
    CommandHandler("sticker", sticker_menu),
    MessageHandler(filters.Sticker.ALL & filters.ChatType.PRIVATE, sticker_to_png),
    CallbackQueryHandler(handle_callback, pattern="^sticker_")
]