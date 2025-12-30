import logging
import os
import re
import json
import asyncio
import paramiko
from telegram import Update, InlineKeyboardButton, InlineKeyboardMarkup, ForceReply
from telegram.ext import ContextTypes, CommandHandler, CallbackQueryHandler, MessageHandler, filters
from config.config import Config
from utils.permissions import is_admin

logger = logging.getLogger(__name__)

async def safe_callback_answer(query, text=None, show_alert=False):
    """Helper to safely answer callback queries ignoring timeouts."""
    try:
        await query.answer(text=text, show_alert=show_alert)
    except Exception as e:
        logger.warning(f"Callback answer failed: {e}")

IP_HISTORY_FILE = "data/ip_history.json"

if not os.path.exists("data"):
    os.makedirs("data")

def get_stored_ips():
    if os.path.exists(IP_HISTORY_FILE):
        try:
            with open(IP_HISTORY_FILE, 'r') as f:
                return json.load(f)
        except:
            pass
    return {"v4": None, "v6": None}

def save_stored_ips(ips):
    try:
        with open(IP_HISTORY_FILE, 'w') as f:
            json.dump(ips, f)
    except Exception as e:
        logger.error(f"Failed to save IP history: {e}")

def get_router_ips():
    """Helper to get current IPv4 and IPv6 from router."""
    v4 = None
    v6 = None

    iface_list = ["wan", "wan_6", "wan6"]
    
    for iface in iface_list:
        try:
            ubus_cmd = f"ubus call network.interface.{iface} status"
            ubus_res = ssh_exec(ubus_cmd)
            
            if ubus_res:
                data = json.loads(ubus_res)
                
                if not v4 and "ipv4-address" in data and len(data["ipv4-address"]) > 0:
                    addr = data["ipv4-address"][0]["address"]
                    if "." in addr:
                        v4 = addr
                
                if not v6 and "ipv6-address" in data:
                    for addr_obj in data["ipv6-address"]:
                        addr = addr_obj["address"]
                        if ":" in addr and not addr.startswith("fe80"):
                            v6 = addr
                            break
                
                if not v6 and "ipv6-prefix-assignment" in data:
                    for prefix in data["ipv6-prefix-assignment"]:
                        if "address" in prefix and not prefix["address"].startswith("fe80"):
                            v6 = prefix["address"]
                            break
        except Exception as e:
            continue
            
    
    if not v4:
        cmd_v4 = "/usr/bin/curl -4 -s --max-time 5 icanhazip.com || /usr/bin/curl -4 -s --max-time 5 ifconfig.me"
        v4_res = ssh_exec(cmd_v4)
        if v4_res and v4_res.strip() and " " not in v4_res:
             v4 = v4_res.strip()

    if not v6:
        cmd_v6 = "/usr/bin/curl -6 -s --max-time 5 icanhazip.com || /usr/bin/curl -6 -s --max-time 5 ifconfig.co"
        v6_res = ssh_exec(cmd_v6)
        if v6_res and v6_res.strip() and " " not in v6_res:
             v6 = v6_res.strip()
    
    if v4 and (len(v4) > 15 or "." not in v4): v4 = None
    if v6 and ":" not in v6: v6 = None
    
    return v4, v6

async def check_ip_job(context: ContextTypes.DEFAULT_TYPE):
    """Job to check for IP changes."""
    try:
        current_v4, current_v6 = get_router_ips()

        if not current_v4 and not current_v6:
            return

        stored = get_stored_ips()
        changed = False
        msg = "🚨 **公网 IP 变动通知**\n-------------------\n"

        if current_v4 and current_v4 != stored.get("v4"):
            msg += f"🔴 IPv4: `{current_v4}`\n(旧: {stored.get('v4', '未知')})\n"
            stored["v4"] = current_v4
            changed = True
        
        if current_v6 and current_v6 != stored.get("v6"):
            msg += f"🔵 IPv6: `{current_v6}`\n(旧: {stored.get('v6', '未知')})\n"
            stored["v6"] = current_v6
            changed = True

        if changed:
            save_stored_ips(stored)
            try:
                await context.bot.send_message(chat_id=Config.ADMIN_ID, text=msg, parse_mode="Markdown")
            except Exception as e:
                logger.error(f"Failed to send IP notification: {e}")
    except Exception as e:
        logger.error(f"Error in check_ip_job: {e}")

def init_plugin(application):
    """Registers the IP check job."""
    if application.job_queue:
        application.job_queue.run_repeating(check_ip_job, interval=60, first=10)
        logger.info("IP Monitor Job registered.")

def ssh_exec(command):
    host = Config.OPENWRT_HOST
    port = Config.OPENWRT_PORT
    user = Config.OPENWRT_USER
    pwd = Config.OPENWRT_PASS
    
    if not pwd:
        logger.error("SSH password not set in .env")
        return None

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        client.connect(host, port=port, username=user, password=pwd, timeout=5)
        stdin, stdout, stderr = client.exec_command(command)
        res = stdout.read().decode('utf-8')
        client.close()
        return res
    except Exception as e:
        logger.error(f"SSH Error: {e}")
        return None

async def wrt_menu(update: Update, context: ContextTypes.DEFAULT_TYPE):
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
         InlineKeyboardButton("🤖 AI 分析日志", callback_data="wrt_ai_analyze")],
        [InlineKeyboardButton("🔄 重启系统", callback_data="wrt_reboot_confirm")],
        [InlineKeyboardButton("🔙 返回", callback_data="start_main")]
    ]
    reply_markup = InlineKeyboardMarkup(keyboard)
    msg = "📟 **OpenWrt 面板**"
    
    if update.callback_query:
        await safe_callback_answer(update.callback_query)
        await update.callback_query.edit_message_text(msg.replace("**", ""), reply_markup=reply_markup)
    else:
        await update.message.reply_text(msg.replace("**", ""), reply_markup=reply_markup)

async def wrt_show_current_ips(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query, "正在查询 IP...")
    
    current_v4, current_v6 = get_router_ips()
    
    if not current_v4 and not current_v6:
        await query.edit_message_text("❌ 无法获取 IP 地址，请检查网络或 SSH 连接。", 
                                      reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_main")]]))
        return

    msg = "🏠 **当前公网 IP**\n-------------------\n"
    msg += f"🔴 IPv4: `{current_v4}`\n" if current_v4 else "🔴 IPv4: 未检测到\n"
    msg += f"🔵 IPv6: `{current_v6}`\n" if current_v6 else "🔵 IPv6: 未检测到\n"
    
    keyboard = [[InlineKeyboardButton("🔙 返回", callback_data="wrt_main")]]
    await query.edit_message_text(msg, parse_mode="Markdown", reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_scripts_list(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query, "读取脚本列表...")
    
    script_dir = "/root/smart"
    cmd = f"ls {script_dir}/*.sh 2>/dev/null"
    res = ssh_exec(cmd)
    
    if not res:
        await query.edit_message_text(f"目录 {script_dir} 下没有找到脚本。")
        return

    scripts = res.splitlines()
    keyboard = []
    for s in scripts:
        name = os.path.basename(s)
        keyboard.append([InlineKeyboardButton(f"▶️ {name}", callback_data=f"wrt_run_{s}")])
        
    keyboard.append([InlineKeyboardButton("🔙 返回", callback_data="wrt_main")])
    await query.edit_message_text(f"📂 脚本列表 ({script_dir}):\n点击即可立即运行。", reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_run_script(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    script_path = query.data.replace("wrt_run_", "")
    
    await safe_callback_answer(query, "正在运行脚本...", show_alert=True)
    await query.edit_message_text(f"⏳ 正在执行: {script_path}\n请稍候...")
    

    res = ssh_exec(script_path)
    
    if res and len(res) > 3000:
        res = res[:3000] + "\n... (输出过长已截断)"
    
    result_text = f"✅ 执行完成: {script_path}\n\n📝 输出:\n{res}" if res else f"✅ 执行完成 (无输出)"
    
    keyboard = [[InlineKeyboardButton("🔙 返回脚本列表", callback_data="wrt_scripts_list")]]
    await query.edit_message_text(result_text, reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_status(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query, "正在通过 SSH 获取数据...")
    
    cmd = "uptime && free -m && [ -f /sys/class/thermal/thermal_zone0/temp ] && cat /sys/class/thermal/thermal_zone0/temp || echo 0"
    res = ssh_exec(cmd)
    
    if not res:
        await query.edit_message_text("无法通过 SSH 连接到路由器，请检查配置。")
        return

    lines = res.splitlines()
    uptime_info = lines[0]
    mem_info = lines[2] 
    for l in lines:
        if "Mem:" in l:
            mem_parts = l.split()
            mem_total, mem_used = mem_parts[1], mem_parts[2]
            break
    else:
        mem_total, mem_used = "0", "0"
        
    temp_raw = lines[-1]
    temp = f"{int(temp_raw)/1000:.1f}°C" if temp_raw.isdigit() and int(temp_raw) > 0 else "N/A"

    txt = (
        f"📟 OpenWrt 状态\n"
        f"-------------------\n"
        f"⏱ 运行时间: {uptime_info.split('up')[1].split(',')[0].strip()}\n"
        f"📈 系统负载: {uptime_info.split('load average:')[1].strip()}\n"
        f"🧠 内存占用: {mem_used}MB / {mem_total}MB\n"
        f"🌡 核心温度: {temp}"
    )
    
    keyboard = [[InlineKeyboardButton("🔙 返回", callback_data="wrt_main")]]
    await query.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_devices(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query, "获取设备列表中...")
    
    cmd = "cat /tmp/dhcp.leases"
    res = ssh_exec(cmd)
    
    if not res:
        await query.edit_message_text("获取失败。")
        return

    txt = "📱 当前联网设备 (DHCP):\n-------------------\n"
    lines = res.splitlines()
    for line in lines:
        parts = line.split()
        if len(parts) >= 4:
            ip, name = parts[2], parts[3]
            txt += f"• {name} ({ip})\n"
            
    if not lines:
        txt += "没有活跃设备。"

    keyboard = [[InlineKeyboardButton("🔙 返回", callback_data="wrt_main")]]
    await query.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_reboot_confirm(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query)
    keyboard = [
        [InlineKeyboardButton("✅ 确认重启", callback_data="wrt_reboot_do"),
         InlineKeyboardButton("❌ 取消", callback_data="wrt_main")]
    ]
    await query.edit_message_text("⚠️ 确认要重启路由器吗？\n重启期间网络将会中断。", reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_reboot_do(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query, "指令已发送")
    await query.edit_message_text("🚀 正在重启路由器，请等待网络恢复...")
    ssh_exec("reboot")

async def wrt_net_menu(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query)
    keyboard = [
        [InlineKeyboardButton("📶 Ping 测试", callback_data="wrt_net_ping_ask"),
         InlineKeyboardButton("📍 路由追踪", callback_data="wrt_net_trace_ask")],
        [InlineKeyboardButton("🔍 DNS 查询", callback_data="wrt_net_nslookup_ask"),
         InlineKeyboardButton("🌐 HTTP 检测", callback_data="wrt_net_curl_ask")],
        [InlineKeyboardButton("🏠 公网 IP", callback_data="wrt_net_myip"),
         InlineKeyboardButton("🔙 返回", callback_data="wrt_main")]
    ]
    await query.edit_message_text("🌐 **网络连接测试**\n请选择测试工具：", reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_net_ping_ask(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query)
    await context.bot.send_message(
        chat_id=update.effective_chat.id,
        text="📡 请回复此消息输入要 Ping 的地址/域名：\n(例如: 8.8.8.8 或 google.com)",
        reply_markup=ForceReply(selective=True)
    )

async def wrt_net_trace_ask(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query)
    await context.bot.send_message(
        chat_id=update.effective_chat.id,
        text="📍 请回复此消息输入要追踪的目标地址：\n(例如: 1.1.1.1)",
        reply_markup=ForceReply(selective=True)
    )

async def wrt_net_nslookup_ask(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query)
    await context.bot.send_message(
        chat_id=update.effective_chat.id,
        text="🔍 请回复此消息输入要查询的域名：\n(例如: google.com)",
        reply_markup=ForceReply(selective=True)
    )

async def wrt_net_curl_ask(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query)
    await context.bot.send_message(
        chat_id=update.effective_chat.id,
        text="🌐 请回复此消息输入要检测的 URL：\n(例如: https://www.google.com)",
        reply_markup=ForceReply(selective=True)
    )

async def wrt_net_myip(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query, "正在查询...")
    
    cmd = "curl -s --connect-timeout 5 ifconfig.me || curl -s --connect-timeout 5 icanhazip.com"
    res = ssh_exec(cmd)
    
    if not res:
        res = "❌ 获取失败"
        
    await query.edit_message_text(f"🏠 **当前公网 IP**:\n`{res}`", parse_mode="Markdown", 
                                  reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_net_menu")]]))

async def handle_wrt_message(update: Update, context: ContextTypes.DEFAULT_TYPE):
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

async def handle_callback(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    data = query.data
    
    if data == "wrt_main":
        await wrt_menu(update, context)
    elif data == "wrt_status":
        await wrt_status(update, context)
    elif data == "wrt_show_current_ips":
        await wrt_show_current_ips(update, context)
    elif data == "wrt_devices":
        await wrt_devices(update, context)
    elif data == "wrt_reboot_confirm":
        await wrt_reboot_confirm(update, context)
    elif data == "wrt_reboot_do":
        await wrt_reboot_do(update, context)
    elif data == "wrt_net_menu":
        await wrt_net_menu(update, context)
    elif data == "wrt_net_ping_ask":
        await wrt_net_ping_ask(update, context)
    elif data == "wrt_net_trace_ask":
        await wrt_net_trace_ask(update, context)
    elif data == "wrt_net_nslookup_ask":
        await wrt_net_nslookup_ask(update, context)
    elif data == "wrt_net_curl_ask":
        await wrt_net_curl_ask(update, context)
    elif data == "wrt_net_myip":
        await wrt_net_myip(update, context)
    elif data == "wrt_scripts_list":
        await wrt_scripts_list(update, context)
    elif data.startswith("wrt_run_"):
        await wrt_run_script(update, context)

handlers = [
    CommandHandler("wrt", wrt_menu),
    CallbackQueryHandler(handle_callback, pattern="^wrt_"),
    MessageHandler(filters.TEXT & (~filters.COMMAND) & filters.REPLY, handle_wrt_message)
]
