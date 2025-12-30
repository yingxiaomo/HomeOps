import logging
import io
import re
import asyncio
import aiohttp
import warnings
warnings.filterwarnings("ignore", category=FutureWarning, module="google.generativeai")
import google.generativeai as genai
from PIL import Image
from telegram import Update, InlineKeyboardButton, InlineKeyboardMarkup
from telegram.ext import ContextTypes, CommandHandler, CallbackQueryHandler, MessageHandler, filters
from config.config import Config
from utils.permissions import has_permission, is_admin
import paramiko
import json

logger = logging.getLogger(__name__)


class GeminiClient:
    def __init__(self):
        self.api_keys = Config.GEMINI_API_KEYS

        self.models = [
            'gemini-3-pro-preview', 
            'gemini-2.5-pro', 
            'gemini-3-flash-preview', 
            'gemini-2.5-flash', 
            'gemini-2.0-flash'
        ]
        self.current_key_index = 0
        self._configure_current_key()

    def _configure_current_key(self):
        if self.api_keys:
            genai.configure(api_key=self.api_keys[self.current_key_index])

    def _rotate_key(self):
        if not self.api_keys or len(self.api_keys) <= 1: return False
        self.current_key_index = (self.current_key_index + 1) % len(self.api_keys)
        self._configure_current_key()
        return True

    async def generate_content(self, prompt, image=None):
        if not self.api_keys: raise Exception("No API Keys")
        last_error = None
        for model_name in self.models:
            start_key_index = self.current_key_index
            while True:
                try:
                    logger.info(f"Attempting with Model: {model_name}, Key Index: {self.current_key_index}")
                    
                    model = genai.GenerativeModel(model_name)
                    if image: response = await model.generate_content_async([prompt, image])
                    else: response = await model.generate_content_async(prompt)
                    return response
                except Exception as e:
                    last_error = e
     
                    if not self._rotate_key() or self.current_key_index == start_key_index: 

                        break
        raise last_error

    async def send_chat_message(self, chat_session, content):

        # TODO: 
        try: return await chat_session.send_message_async(content)
        except Exception as e:
            self._rotate_key()
            raise e

gemini_client = GeminiClient()

async def clash_api_patch(payload):
    """Internal helper to change clash config via API."""
    url = f"{Config.OPENCLASH_API_URL}/configs"
    headers = {"Authorization": f"Bearer {Config.OPENCLASH_API_SECRET}"} if Config.OPENCLASH_API_SECRET else {}
    try:
        async with aiohttp.ClientSession() as session:
            async with session.patch(url, json=payload, headers=headers, timeout=5) as resp:
                return resp.status in [202, 204]
    except:
        return False

async def clash_api_get_config():
    url = f"{Config.OPENCLASH_API_URL}/configs"
    headers = {"Authorization": f"Bearer {Config.OPENCLASH_API_SECRET}"} if Config.OPENCLASH_API_SECRET else {}
    try:
        async with aiohttp.ClientSession() as session:
            async with session.get(url, headers=headers, timeout=5) as resp:
                return await resp.json()
    except:
        return None

def ssh_exec_simple(cmd):
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        client.connect(Config.OPENWRT_HOST, port=Config.OPENWRT_PORT, username=Config.OPENWRT_USER, password=Config.OPENWRT_PASS, timeout=5)
        stdin, stdout, stderr = client.exec_command(cmd)
        res = stdout.read().decode('utf-8').strip()
        client.close()
        return res
    except:
        return ""


async def ai_mode_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    if not has_permission(update.effective_user.id, 'ai'):
        await query.answer("⛔ 无权使用", show_alert=True); return
    await query.answer()
    context.user_data['ai_mode'] = True
    try:
        chat = genai.GenerativeModel('gemini-3-pro-preview').start_chat(history=[])
        context.user_data['ai_chat_session'] = chat
    except Exception as e:
        await query.edit_message_text(f"❌ 初始化失败: {e}"); return
    await query.edit_message_text("🧠 **已进入 AI 沉浸模式 (3.0 Pro)**\n可以直接对话或发送图片。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🚪 退出", callback_data="ai_mode_exit")]]), parse_mode="Markdown")

async def ai_mode_exit(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    context.user_data['ai_mode'] = False
    await query.answer("已退出")
    from plugins.start import start
    await start(update, context)

from telegram.error import BadRequest

async def safe_edit_text(message, text, **kwargs):
    """Helper to edit text with Markdown fallback."""
    try:
        await message.edit_text(text, **kwargs)
    except BadRequest as e:
        if "can't parse entities" in str(e).lower():
            kwargs.pop('parse_mode', None)
            await message.edit_text(text, **kwargs)
        else:
            raise e

async def ai_message_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    if not context.user_data.get('ai_mode'): return
    msg = update.message
    status_msg = await msg.reply_text("🤔 思考中...")
    try:
        chat = context.user_data.get('ai_chat_session')
        if not chat: chat = genai.GenerativeModel('gemini-3-pro-preview').start_chat(history=[]); context.user_data['ai_chat_session'] = chat
        
        parts = [msg.text or msg.caption or ""]
        if msg.photo:
            img_bytes = io.BytesIO()
            await ((await msg.photo[-1].get_file()).download_to_memory(img_bytes))
            parts.append(Image.open(img_bytes))
            
        response = await gemini_client.send_chat_message(chat, parts)
        await safe_edit_text(status_msg, response.text[:4000], parse_mode="Markdown", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🚪 退出", callback_data="ai_mode_exit")]]))
    except Exception as e:
        await status_msg.edit_text(f"❌ 错误: {e}")


async def analyze_target_logs(update: Update, log_type: str):
    user = update.effective_user
    if not is_admin(user.id):
        await update.effective_message.reply_text("⛔ 仅限管理员使用。"); return
    
    query = update.callback_query
    status_msg = await (query.message if query else update.message).reply_text(f"🔍 启动 {log_type} 深度自动化诊断...")
    
    original_log_level = "info"
    
    if log_type == "OpenClash":
        config = await clash_api_get_config()
        original_log_level = "info"
        if config:
            original_log_level = config.get("log-level", "info")
            if original_log_level != "debug":
                await status_msg.edit_text(f"⚙️ 当前级别为 {original_log_level}，正在临时切换至 debug 以获取完整握手细节...")
                await clash_api_patch({"log-level": "debug"})
                await asyncio.sleep(5) 
        
        await status_msg.edit_text("📡 正在全量采集多源日志 (每项深度 100 行)...")
        
        diag_cmd = (
            "echo '--- [KERNEL LOG (DEBUG MODE)] ---'; tail -n 100 /tmp/openclash.log 2>/dev/null; "
            "echo '--- [STARTUP/PLUGIN LOG] ---'; tail -n 100 /tmp/openclash_start.log 2>/dev/null; "
            "echo '--- [SYSTEM SYSLOG] ---'; logread | grep -E -i 'clash|openclash' | tail -n 100; "
            "echo '--- [NETWORK STATUS] ---'; ubus call network.interface.wan status | grep -E 'up|address|pending'"
        )
        logs = ssh_exec_simple(diag_cmd)
        
        if config and original_log_level != "debug":
            await clash_api_patch({"log-level": original_log_level})

        prompt = (
            f"你是 OpenClash 专家。用户平时使用的日志等级是 '{original_log_level}'，但为了本次诊断，"
            "我已临时将等级提升至 'debug' 并抓取了以下 4 个维度的聚合数据。请进行深度分析：\n\n"
            "分析要求：\n"
            "1. 检查 KERNEL 部分是否有节点握手失败、TLS 证书问题或 DNS 查询超时。\n"
            "2. 检查 STARTUP 部分是否有配置文件生成失败、订阅下载错误或内核权限问题。\n"
            "3. 检查 SYSTEM 部分是否有路由器内存不足 (OOM) 或网络接口重置的情况。\n"
            "4. 综合判断当前的上网故障原因，并给出中文建议。\n\n"
            f"诊断聚合数据：\n{logs}"
        )
        back_cb = "clash_main"
    else:
        logs = ssh_exec_simple("logread | tail -n 100")
        prompt = f"分析 OpenWrt 系统日志：\n{logs}"
        back_cb = "wrt_main"

    if not logs:
        await status_msg.edit_text("❌ 采集失败，请检查 SSH 权限。"); return

    await status_msg.edit_text("🤖 正在利用 Gemini 3.0 Pro 进行多维度联合分析...")
    try:
        response = await gemini_client.generate_content(prompt)
        await safe_edit_text(status_msg, f"📋 **AI {log_type} 综合诊断报告**\n-------------------\n{response.text[:3800]}", 
                                   parse_mode="Markdown", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data=back_cb)]]))
    except Exception as e:
        await status_msg.edit_text(f"❌ 分析失败: {e}")


async def analyze_logs(u, c): await analyze_target_logs(u, "OpenWrt")
async def analyze_clash_logs(u, c): await analyze_target_logs(u, "OpenClash")

async def handle_callback(update: Update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    if q.data == "ai_mode_start": await ai_mode_start(update, context)
    elif q.data == "ai_mode_exit": await ai_mode_exit(update, context)
    elif q.data == "wrt_ai_analyze": await analyze_logs(update, context)
    elif q.data == "wrt_ai_clash": await analyze_clash_logs(update, context)

handlers = [
    CommandHandler("ai", ai_mode_start),
    MessageHandler((filters.TEXT | filters.PHOTO) & (~filters.COMMAND), ai_message_handler),
    CallbackQueryHandler(handle_callback, pattern="^(ai_|wrt_ai_)")
]
