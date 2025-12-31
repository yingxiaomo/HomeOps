from telegram import InlineKeyboardButton, InlineKeyboardMarkup, ForceReply
from telegram.ext import ContextTypes
from .helpers import safe_callback_answer
from .connection import ssh_exec
import re

async def wrt_net_menu(update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query)
    keyboard = [
        [InlineKeyboardButton("⚡ 快速诊断", callback_data="wrt_net_quick"),
         InlineKeyboardButton("✍️ 手动测试", callback_data="wrt_net_manual")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_main")]
    ]
    await query.edit_message_text("🌐 **网络连接测试**\n请选择测试模式：", reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_net_manual(update, context):
    query = update.callback_query
    await safe_callback_answer(query)
    keyboard = [
        [InlineKeyboardButton("📶 Ping 测试", callback_data="wrt_net_ping_ask"),
         InlineKeyboardButton("📍 路由追踪", callback_data="wrt_net_trace_ask")],
        [InlineKeyboardButton("🔍 DNS 查询", callback_data="wrt_net_nslookup_ask"),
         InlineKeyboardButton("🌐 HTTP 检测", callback_data="wrt_net_curl_ask")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_net_menu")]
    ]
    await query.edit_message_text("✍️ **手动测试**\n请选择工具并输入目标：", reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_net_quick(update, context):
    query = update.callback_query
    await safe_callback_answer(query)
    keyboard = [
        [InlineKeyboardButton("📶 Ping 网关", callback_data="wrt_net_run_ping_gateway"),
         InlineKeyboardButton("📶 Ping 百度", callback_data="wrt_net_run_ping_baidu")],
        [InlineKeyboardButton("📶 Ping Google", callback_data="wrt_net_run_ping_google"),
         InlineKeyboardButton("📶 Ping DNS (8.8.8.8)", callback_data="wrt_net_run_ping_dns")],
        [InlineKeyboardButton("📍 Trace Google", callback_data="wrt_net_run_trace_google"),
         InlineKeyboardButton("🔍 查 Google IP", callback_data="wrt_net_run_ns_google")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_net_menu")]
    ]
    await query.edit_message_text("⚡ **快速诊断**\n一键执行常用网络测试：", reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_net_run_quick(update, context):
    query = update.callback_query
    data = query.data
    await safe_callback_answer(query, "正在执行测试...")
    
    cmd = ""
    title = ""
    
    if data == "wrt_net_run_ping_gateway":
        # Try to find gateway
        gw_cmd = "ip route | grep default | awk '{print $3}' | head -n 1"
        gw = ssh_exec(gw_cmd)
        gw = gw.strip() if gw else "192.168.1.1"
        cmd = f"ping -c 4 -w 5 {gw}"
        title = f"Ping Gateway ({gw})"
    elif data == "wrt_net_run_ping_baidu":
        cmd = "ping -c 4 -w 5 www.baidu.com"
        title = "Ping Baidu"
    elif data == "wrt_net_run_ping_google":
        cmd = "ping -c 4 -w 5 www.google.com"
        title = "Ping Google"
    elif data == "wrt_net_run_ping_dns":
        cmd = "ping -c 4 -w 5 8.8.8.8"
        title = "Ping 8.8.8.8"
    elif data == "wrt_net_run_trace_google":
        cmd = "traceroute -I -m 15 -w 2 -q 1 -n www.google.com 2>/dev/null || traceroute -m 15 -w 2 -q 1 -n www.google.com"
        title = "Trace Google"
    elif data == "wrt_net_run_ns_google":
        cmd = "nslookup www.google.com"
        title = "Nslookup Google"
        
    await query.edit_message_text(f"⏳ 正在执行 {title}...")
    res = ssh_exec(cmd)
    if not res:
        res = "❌ 执行失败或无输出"
    if len(res) > 3000:
        res = res[:3000] + "\n...(truncated)"
        
    keyboard = [[InlineKeyboardButton("🔙 返回快速诊断", callback_data="wrt_net_quick")]]
    await query.edit_message_text(f"📝 **{title} 结果**:\n```\n{res}\n```", parse_mode="Markdown", reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_net_ping_ask(update, context):
    query = update.callback_query
    await safe_callback_answer(query)
    await context.bot.send_message(
        chat_id=update.effective_chat.id,
        text="📡 请回复此消息输入要 Ping 的地址/域名：\n(例如: 8.8.8.8 或 google.com)",
        reply_markup=ForceReply(selective=True)
    )

async def wrt_net_trace_ask(update, context):
    query = update.callback_query
    await safe_callback_answer(query)
    await context.bot.send_message(
        chat_id=update.effective_chat.id,
        text="📍 请回复此消息输入要追踪的目标地址：\n(例如: 1.1.1.1)",
        reply_markup=ForceReply(selective=True)
    )

async def wrt_net_nslookup_ask(update, context):
    query = update.callback_query
    await safe_callback_answer(query)
    await context.bot.send_message(
        chat_id=update.effective_chat.id,
        text="🔍 请回复此消息输入要查询的域名：\n(例如: google.com)",
        reply_markup=ForceReply(selective=True)
    )

async def wrt_net_curl_ask(update, context):
    query = update.callback_query
    await safe_callback_answer(query)
    await context.bot.send_message(
        chat_id=update.effective_chat.id,
        text="🌐 请回复此消息输入要检测的 URL：\n(例如: https://www.google.com)",
        reply_markup=ForceReply(selective=True)
    )

async def wrt_net_myip(update, context):
    query = update.callback_query
    await safe_callback_answer(query, "正在查询...")
    cmd = "curl -s --connect-timeout 5 ifconfig.me || curl -s --connect-timeout 5 icanhazip.com"
    res = ssh_exec(cmd)
    if not res:
        res = "❌ 获取失败"
    await query.edit_message_text(f"🏠 **当前公网 IP**:\n`{res}`", parse_mode="Markdown", 
                                  reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_net_menu")]]))

async def handle_wrt_message(update, context: ContextTypes.DEFAULT_TYPE):
    from .firewall import handle_fw_wizard
    from .adg import handle_adg_wizard
    if await handle_adg_wizard(update, context):
        return
    if await handle_fw_wizard(update, context):
        return
    if not update.message.reply_to_message:
        return
    prompt = update.message.reply_to_message.text
    user_input = update.message.text.strip()
    if not re.match(r'^[a-zA-Z0-9\.\-\_\:\/\s]+$', user_input):
        await update.message.reply_text("❌ 输入包含非法字符 (如 ; | & 等)。")
        return
    cmd = ""
    tool_name = ""
    if "要 Ping 的地址" in prompt:
        tool_name = "Ping"
        cmd = f"ping -c 4 -w 5 {user_input}"
    elif "要追踪的目标" in prompt:
        tool_name = "Traceroute"
        cmd = f"traceroute -I -m 15 -w 2 -q 1 -n {user_input} 2>/dev/null || traceroute -m 15 -w 2 -q 1 -n {user_input}"
    elif "要查询的域名" in prompt:
        tool_name = "Nslookup"
        cmd = f"nslookup {user_input}"
    elif "要检测的 URL" in prompt:
        tool_name = "Curl"
        cmd = f"curl -I -s -w 'Response Code: %{{http_code}}\\nTime: %{{time_total}}s\\n' -o /dev/null {user_input}"
    elif prompt.startswith("➕ 添加端口转发") or prompt.startswith("➕ 添加通信规则"):
        return
    else:
        return
    status_msg = await update.message.reply_text(f"⏳ 正在执行 {tool_name} {user_input}...")
    res = ssh_exec(cmd)
    if not res:
        res = "❌ 执行失败或无输出"
    if len(res) > 3000:
        res = res[:3000] + "\n...(truncated)"
    keyboard = [[InlineKeyboardButton("🔙 返回测试菜单", callback_data="wrt_net_menu")]]
    await status_msg.edit_text(f"📝 **{tool_name} 结果**:\n```\n{res}\n```", parse_mode="Markdown", reply_markup=InlineKeyboardMarkup(keyboard))
