import json
import os
import aiohttp
import asyncio
import logging
from telegram import Update, InlineKeyboardButton, InlineKeyboardMarkup
from telegram.ext import ContextTypes, CommandHandler, CallbackQueryHandler
from config.config import Config
from utils.permissions import is_admin

logger = logging.getLogger(__name__)

_clash_session = None

async def api_request(method, endpoint, json_data=None):
    base_url = Config.OPENCLASH_API_URL
    secret = Config.OPENCLASH_API_SECRET
    headers = {"Authorization": f"Bearer {secret}"} if secret else {}
    global _clash_session
    if _clash_session is None:
        _clash_session = aiohttp.ClientSession()
    url = f"{base_url}{endpoint}"
    try:
        async with _clash_session.request(method, url, headers=headers, json=json_data, proxy=None) as resp:
            if resp.status == 204:
                return True
            return await resp.json()
    except Exception as e:
        logger.error(f"OpenClash API Error [{method} {url}]: {e}")
        return None

async def get_traffic_snapshot():
    base_url = Config.OPENCLASH_API_URL
    secret = Config.OPENCLASH_API_SECRET
    headers = {"Authorization": f"Bearer {secret}"} if secret else {}

    global _clash_session
    if _clash_session is None:
        _clash_session = aiohttp.ClientSession()
    url = f"{base_url}/traffic"
    try:
        async with _clash_session.ws_connect(url, headers=headers, proxy=None, timeout=3.0) as ws:
            msg = await ws.receive_json(timeout=2.0)
            return msg
    except Exception as e:
        logger.error(f"OpenClash WS Error: {e}")
        return None

async def clash_menu(update: Update, context: ContextTypes.DEFAULT_TYPE):
    user = update.effective_user
    if not is_admin(user.id):
        await update.message.reply_text("⛔ 此功能仅限管理员使用。")
        return

    configs = await api_request("GET", "/configs")
    mode = configs.get("mode", "Unknown") if configs else "Unknown"

    keyboard = [
        [InlineKeyboardButton(f"⚙️ 模式: {mode}", callback_data="clash_modes")],
        [InlineKeyboardButton("📊 状态", callback_data="clash_status"),
         InlineKeyboardButton("🌍 节点", callback_data="clash_groups")],
        [InlineKeyboardButton("🧰 工具箱", callback_data="clash_tools")],
        [InlineKeyboardButton("🔙 返回", callback_data="clash_exit")]
    ]
    reply_markup = InlineKeyboardMarkup(keyboard)
    msg = f"🚀 **OpenClash 面板**"
    
    if update.callback_query:
        await update.callback_query.answer()
        await update.callback_query.edit_message_text(msg.replace("**", "").replace("`", ""), reply_markup=reply_markup)
    else:
        await update.message.reply_text(msg.replace("**", "").replace("`", ""), reply_markup=reply_markup)

async def clash_modes(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()
    
    keyboard = [
        [InlineKeyboardButton("Rule (规则)", callback_data="clash_setmode_rule"),
         InlineKeyboardButton("Global (全局)", callback_data="clash_setmode_global")],
        [InlineKeyboardButton("Direct (直连)", callback_data="clash_setmode_direct")],
        [InlineKeyboardButton("Script (脚本)", callback_data="clash_setmode_script")],
        [InlineKeyboardButton("🔙 返回", callback_data="clash_main")]
    ]
    await query.edit_message_text("请选择运行模式:", reply_markup=InlineKeyboardMarkup(keyboard))

async def clash_set_mode(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    mode = query.data.replace("clash_setmode_", "") 
    mode_cap = mode.capitalize()
    
    success = await api_request("PATCH", "/configs", {"mode": mode_cap})
    
    if success:
        await query.answer(f"模式已切换为 {mode_cap}")
        await clash_menu(update, context) 
    else:
        await query.answer("切换失败")

async def clash_tools(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()
    
    configs = await api_request("GET", "/configs")
    current_level = configs.get("log-level", "info") if configs else "unknown"
    debug_btn_text = f"🐛 切换调试模式 (当前: {current_level})"

    keyboard = [
        [InlineKeyboardButton("⚡ 全局节点测速", callback_data="clash_speedtest_all")],
        [InlineKeyboardButton("🤖 AI 分析内核日志", callback_data="wrt_ai_clash")],
        [InlineKeyboardButton(debug_btn_text, callback_data="clash_toggle_debug")],
        [InlineKeyboardButton("♻️ 重载配置 (含清DNS)", callback_data="clash_reload")],
        [InlineKeyboardButton("✂️ 断开所有连接", callback_data="clash_flush_conns")],
        [InlineKeyboardButton("🧹 清除 FakeIP 缓存", callback_data="clash_flush_fakeip")],
        [InlineKeyboardButton("🔙 返回主菜单", callback_data="clash_main")]
    ]
    await query.edit_message_text("🛠 工具箱操作:", reply_markup=InlineKeyboardMarkup(keyboard))

async def clash_toggle_debug(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    
    configs = await api_request("GET", "/configs")
    if not configs:
        await query.answer("无法获取当前配置")
        return

    current_level = configs.get("log-level", "info")
    new_level = "debug" if current_level != "debug" else "info"
    success = await api_request("PATCH", "/configs", {"log-level": new_level})
    
    if success:
        await query.answer(f"日志级别已切换为: {new_level}", show_alert=True)
        await clash_tools(update, context)
    else:
        await query.answer("切换失败")

async def clash_speedtest_all(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer("正在对所有节点进行测速，可能需要几十秒...", show_alert=True)
    
    proxies = await api_request("GET", "/proxies")
    if not proxies:
        return

    targets = []
    for name, info in proxies["proxies"].items():
        if "all" not in info: 
            targets.append(name)
            
    batch_size = 20
    for i in range(0, len(targets), batch_size):
        batch = targets[i:i+batch_size]
        tasks = []
        for node in batch:
            endpoint = f"/proxies/{node}/delay?timeout=3000&url=http://www.gstatic.com/generate_204"
            tasks.append(api_request("GET", endpoint))
        await asyncio.gather(*tasks)
        await asyncio.sleep(0.5)
        
    await query.message.reply_text("✅ 全局测速完成！请进入节点列表查看延迟。")
    await clash_tools(update, context)

async def clash_flush_fakeip(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    success = await api_request("POST", "/cache/fakeip/flush")
    
    if success:
        await query.answer("FakeIP 缓存已清除", show_alert=True)
    else:
        await query.answer("清除失败 (可能未开启 FakeIP)", show_alert=True)
    await clash_menu(update, context)

async def clash_reload(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    success = await api_request("PUT", "/configs?force=true", {"path": "", "payload": ""})

    
    if success:
        await query.answer("配置已重载", show_alert=True)
    else:
        await query.answer("重载请求失败")
    await clash_menu(update, context)

async def clash_flush_conns(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    success = await api_request("DELETE", "/connections")
    
    if success:
        await query.answer("所有连接已断开", show_alert=True)
    else:
        await query.answer("操作失败")
    await clash_menu(update, context)


async def clash_status(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer("Fetching status...")
    
    version_info = await api_request("GET", "/version")

    connections_info = await api_request("GET", "/connections")
    
    traffic_info = None 

    if not version_info:
        await query.edit_message_text("Error: Cannot connect to Clash API.")
        return

    ver = version_info.get('version', 'Unknown')
    premium = "是" if version_info.get('premium', False) else "否"
    
    conn_count = 0
    upload_total = 0
    download_total = 0
    
    if connections_info and "connections" in connections_info:
        conns = connections_info["connections"]
        conn_count = len(conns)
        for c in conns:
            upload_total += c.get('upload', 0)
            download_total += c.get('download', 0)
            
    def fmt_bytes(size):
        power = 2**10
        n = 0
        power_labels = {0 : '', 1: 'K', 2: 'M', 3: 'G', 4: 'T'}
        while size > power:
            size /= power
            n += 1
        return f"{size:.2f} {power_labels[n]}B"

    up_speed = "N/A" 
    down_speed = "N/A"
    
    txt = (
        f"📊 **Clash 状态监控**\n"
        f"-------------------\n"
        f"🛠 版本: {ver}\n"
        f"💎 Premium内核: {premium}\n"
        f"-------------------\n"
        f"🔗 当前活跃连接: {conn_count}\n"
        f"🚀 实时上传: {up_speed}\n"
        f"⏬ 实时下载: {down_speed}\n"
        f"-------------------\n"
        f"📦 当前会话总流量:\n"
        f"   ⬆️ 上传: {fmt_bytes(upload_total)}\n"
        f"   ⬇️ 下载: {fmt_bytes(download_total)}"
    )
    
    keyboard = [[InlineKeyboardButton("🔙 返回主菜单", callback_data="clash_main")]]
    await query.edit_message_text(txt.replace("**", ""), reply_markup=InlineKeyboardMarkup(keyboard))

async def clash_traffic(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer("Fetching traffic...")
    
    data = await get_traffic_snapshot()
    if not data:
        await query.edit_message_text("Error: Cannot fetch traffic.")
        return

    up = data.get("up", 0) / 1024
    down = data.get("down", 0) / 1024
    
    txt = f"Real-time Traffic:\n⬆️ Upload: {up:.2f} KB/s\n⬇️ Download: {down:.2f} KB/s"
    keyboard = [
        [InlineKeyboardButton("Refresh", callback_data="clash_traffic")],
        [InlineKeyboardButton("Back", callback_data="clash_main")]
    ]
    await query.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(keyboard))

async def clash_groups(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()
    
    proxies = await api_request("GET", "/proxies")
    if not proxies or "proxies" not in proxies:
        await query.edit_message_text("Error fetching proxies.")
        return

    groups = []
    exclude_keywords = ["Apple", "Microsoft", "Google", "Telegram", "Steam", "Speedtest", "Reject", "Direct", "Recycle", "Hijacking", "Video", "Media", "AD", "Bybit"]
    
    for name, info in proxies["proxies"].items():

        if info["type"] == "Selector" and name not in ["DIRECT", "REJECT", "GLOBAL"]:
 
            if not any(k.lower() in name.lower() for k in exclude_keywords):
                groups.append(name)
            
    keyboard = []
    row = []
    for g in groups:
        row.append(InlineKeyboardButton(g, callback_data=f"G_{g}"))
        if len(row) == 4:
            keyboard.append(row)
            row = []
    if row:
        keyboard.append(row)
    
    keyboard.append([InlineKeyboardButton("Back", callback_data="clash_main")])
    await query.edit_message_text("请选择一个代理组 (Proxy Group):", reply_markup=InlineKeyboardMarkup(keyboard))

async def clash_list_nodes(update: Update, context: ContextTypes.DEFAULT_TYPE, group_name_override=None):
    query = update.callback_query

    if not group_name_override:
        await query.answer()
    
    if group_name_override:
        group_name = group_name_override
    else:

        group_name = query.data.replace("G_", "")
        
    proxies = await api_request("GET", "/proxies")
    
    if not proxies:
        logger.error("API returned None for /proxies")
        await query.edit_message_text("无法连接到 Clash API。")
        return

    if group_name not in proxies["proxies"]:
        logger.error(f"Group {group_name} not found in proxies keys: {list(proxies['proxies'].keys())}")
        await query.edit_message_text("未找到该组信息。")
        return

    group_info = proxies["proxies"][group_name]
    all_nodes_names = group_info.get("all", [])
    now_selected = group_info.get("now", "")
    node_details = proxies["proxies"]

    keyboard = []
    row = []
    for node in all_nodes_names:
        info = node_details.get(node, {})
        history = info.get("history", [])
        delay = history[-1].get("delay", 0) if history else 0
        
        delay_str = f"({delay}ms)" if delay > 0 else ""
        label = f"✅ {node} {delay_str}" if node == now_selected else f"{node} {delay_str}"
        
        callback_data = f"S_{group_name}|{node}"
        if len(callback_data.encode('utf-8')) > 64:
            logger.warning(f"Callback data too long: {callback_data}")
            
        row.append(InlineKeyboardButton(label, callback_data=callback_data))
        if len(row) == 2:
            keyboard.append(row)
            row = []
    if row:
        keyboard.append(row)
        
    keyboard.append([InlineKeyboardButton("⚡ 一键测速所有节点", callback_data=f"clash_testall_{group_name}")])
    keyboard.append([InlineKeyboardButton("返回组列表", callback_data="clash_groups")])
    await query.edit_message_text(f"当前组: {group_name}\n当前节点: {now_selected}\n请点击选择新节点:", reply_markup=InlineKeyboardMarkup(keyboard))

async def clash_set_node(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    data = query.data.replace("S_", "").split("|")
    group = data[0]
    node = data[1]
    
    success = await api_request("PUT", f"/proxies/{group}", {"name": node})
    
    if success:
        await query.answer(f"Switched to {node}")
        proxies = await api_request("GET", "/proxies")
        group_info = proxies["proxies"][group]
        all_nodes = group_info.get("all", [])
        now_selected = group_info.get("now", "")
        
        keyboard = []
        row = []
        for n in all_nodes:
            label = f"✅ {n}" if n == now_selected else n
            row.append(InlineKeyboardButton(label, callback_data=f"S_{group}|{n}"))
            if len(row) == 2:
                keyboard.append(row)
                row = []
        if row:
            keyboard.append(row)
        keyboard.append([InlineKeyboardButton("Back", callback_data="clash_groups")])
        await query.edit_message_text(f"Current: {now_selected}\nSelect node for {group}:", reply_markup=InlineKeyboardMarkup(keyboard))
    else:
        await query.answer("Failed to switch.")

async def clash_speedtest_all(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    group_name = query.data.replace("clash_testall_", "")
    
    await query.answer("正在测速所有节点，请稍候...")
    
    proxies = await api_request("GET", "/proxies")
    if not proxies or group_name not in proxies["proxies"]:
        return

    all_nodes = proxies["proxies"][group_name].get("all", [])
    
    async def test_node(node_name):
        endpoint = f"/proxies/{node_name}/delay?timeout=3000&url=http://www.gstatic.com/generate_204"
        await api_request("GET", endpoint)

    tasks = [test_node(name) for name in all_nodes]
    await asyncio.gather(*tasks)
    
    await query.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(keyboard))

async def handle_callback(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    data = query.data
    logger.info(f"Received callback data: {data}")
    
    if data == "clash_main":
        await clash_menu(update, context)
    elif data == "clash_exit":
        try:
            if _clash_session:
                await _clash_session.close()
        except Exception:
            pass
        _clash_session = None
        k = [
            [InlineKeyboardButton("🧠 AI 助手", callback_data="ai_mode_start")],
            [InlineKeyboardButton("🚀 OpenClash", callback_data="clash_main"),
             InlineKeyboardButton("📟 OpenWrt", callback_data="wrt_main")],
            [InlineKeyboardButton("📧 临时邮箱", callback_data="mail_main"),
             InlineKeyboardButton("🖼️ 贴纸转换", callback_data="sticker_main")]
        ]
        await query.edit_message_text("🏠 HomeOps 控制台", reply_markup=InlineKeyboardMarkup(k))
    elif data == "clash_status":
        await clash_status(update, context)
    elif data == "clash_traffic":
        await clash_traffic(update, context)
    elif data == "clash_groups":
        await clash_groups(update, context)
    elif data.startswith("G_"):
        await clash_list_nodes(update, context)
    elif data.startswith("S_"):
        await clash_set_node(update, context)
    elif data.startswith("clash_testall_"):
        await clash_test_all_nodes(update, context)
    elif data == "clash_modes":
        await clash_modes(update, context)
    elif data.startswith("clash_setmode_"):
        await clash_set_mode(update, context)
    elif data == "clash_tools":
        await clash_tools(update, context)
    elif data == "clash_reload":
        await clash_reload(update, context)
    elif data == "clash_speedtest_all":
        await clash_speedtest_all(update, context)
    elif data == "clash_flush_conns":
        await clash_flush_conns(update, context)
    elif data == "clash_flush_fakeip":
        await clash_flush_fakeip(update, context)
    elif data == "clash_toggle_debug":
        await clash_toggle_debug(update, context)

handlers = [
    CommandHandler("clash", clash_menu),
    CallbackQueryHandler(handle_callback, pattern=r"^(clash_|G_|S_)")
]
