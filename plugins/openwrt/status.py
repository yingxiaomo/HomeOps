from telegram import InlineKeyboardButton, InlineKeyboardMarkup
from telegram.ext import ContextTypes
from .helpers import safe_callback_answer
from .connection import ssh_exec

async def wrt_status(update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query, "正在通过 SSH 获取数据...")
    cmd = "uptime && free -m && [ -f /sys/class/thermal/thermal_zone0/temp ] && cat /sys/class/thermal/thermal_zone0/temp || echo 0"
    res = ssh_exec(cmd)
    if not res:
        await query.edit_message_text("无法通过 SSH 连接到路由器，请检查配置。", 
                                      reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_main")]]))
        return
    lines = res.splitlines()
    uptime_info = lines[0]
    mem_total, mem_used = "0", "0"
    for l in lines:
        if "Mem:" in l:
            mem_parts = l.split()
            mem_total, mem_used = mem_parts[1], mem_parts[2]
            break
    temp_raw = lines[-1] if lines else "0"
    temp = f"{int(temp_raw)/1000:.1f}°C" if temp_raw.isdigit() and int(temp_raw) > 0 else "N/A"
    txt = (
        f"📟 OpenWrt 状态\n"
        f"-------------------\n"
        f"⏱ 运行时间: {uptime_info.split('up')[1].split(',')[0].strip()}\n"
        f"📈 系统负载: {uptime_info.split('load average:')[1].strip()}\n"
        f"🧠 内存占用: {mem_used}MB / {mem_total}MB\n"
        f"🌡 核心温度: {temp}"
    )
    keyboard = [
        [InlineKeyboardButton("🛠 服务管理", callback_data="wrt_services_menu"),
         InlineKeyboardButton("🧹 清理内存", callback_data="wrt_drop_caches")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_main")]
    ]
    await query.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_show_current_ips(update, context: ContextTypes.DEFAULT_TYPE):
    from .ip_monitor import get_router_ips
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

async def wrt_reboot_confirm(update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query)
    keyboard = [
        [InlineKeyboardButton("✅ 确认重启", callback_data="wrt_reboot_do"),
         InlineKeyboardButton("❌ 取消", callback_data="wrt_main")]
    ]
    await query.edit_message_text("⚠️ 确认要重启路由器吗？\n重启期间网络将会中断。", reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_reboot_do(update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query, "指令已发送")
    await query.edit_message_text("🚀 正在重启路由器，请等待网络恢复...")
    ssh_exec("reboot")

async def wrt_services_menu(update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query, "正在查询服务状态...")
    
    services = ["network", "firewall", "dnsmasq", "uhttpd"]
    
    keyboard = []
    for svc in services:
        keyboard.append([
            InlineKeyboardButton(f"🔄 重启 {svc}", callback_data=f"wrt_svc_restart_{svc}")
        ])
    
    keyboard.append([InlineKeyboardButton("🔙 返回", callback_data="wrt_status")])
    await query.edit_message_text("🛠 **服务管理**\n请选择要操作的服务：", reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_svc_action(update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    data = query.data
    svc = data.split("_")[-1]
    
    await safe_callback_answer(query, f"正在重启 {svc}...")
    await query.edit_message_text(f"⏳ 正在重启 {svc}，请稍候...")
    
    cmd = f"/etc/init.d/{svc} restart"
    ssh_exec(cmd)
    
    await query.edit_message_text(f"✅ {svc} 重启指令已发送。", 
                                  reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回服务列表", callback_data="wrt_services_menu")]]))

async def wrt_drop_caches(update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await safe_callback_answer(query, "正在清理内存...")
    ssh_exec("sync && echo 3 > /proc/sys/vm/drop_caches")
    await wrt_status(update, context)
