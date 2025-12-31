import asyncio
import base64
import json
import logging
from typing import List, Dict, Optional
import aiohttp
from telegram import InlineKeyboardButton, InlineKeyboardMarkup, ForceReply
from telegram.ext import ContextTypes
from config.config import Config
from .helpers import safe_callback_answer
from .connection import ssh_exec

logger = logging.getLogger(__name__)

_adg_session: Optional[aiohttp.ClientSession] = None
_adg_cookies: Optional[aiohttp.CookieJar] = None

def _adg_base_url():
    url = getattr(Config, "ADG_URL", None) or f"http://{Config.OPENWRT_HOST}:3000"
    return url.rstrip("/")

async def _ensure_session():
    global _adg_session
    if _adg_session is None:
        _adg_session = aiohttp.ClientSession()
    return _adg_session

async def adg_login():
    user = getattr(Config, "ADG_USER", None)
    pwd = getattr(Config, "ADG_PASS", None)
    token = getattr(Config, "ADG_TOKEN", None)
    if not (user and pwd) and not token:
        return True
    try:
        s = await _ensure_session()
        if token:
            # Token-based auth: attach Authorization header on each request
            return True
        payload = {"name": user, "password": pwd}
        async with s.post(f"{_adg_base_url()}/control/login", json=payload) as resp:
            ok = (resp.status == 200)
            if not ok:
                logger.error(f"ADG login failed: HTTP {resp.status}")
            return ok
    except Exception as e:
        logger.error(f"ADG login error: {e}")
        return False

async def adg_api_request(method: str, endpoint: str, json_data=None):
    try:
        s = await _ensure_session()
        headers = {}
        token = getattr(Config, "ADG_TOKEN", None)
        if token:
            headers["Authorization"] = f"Bearer {token}"
        else:
            user = getattr(Config, "ADG_USER", None)
            pwd = getattr(Config, "ADG_PASS", None)
            if user and pwd:
                basic = base64.b64encode(f"{user}:{pwd}".encode("utf-8")).decode("ascii")
                headers["Authorization"] = f"Basic {basic}"
            else:
                await adg_login()
        base = _adg_base_url()
        headers.setdefault("Origin", base)
        headers.setdefault("Referer", base + "/")
        headers.setdefault("X-Requested-With", "XMLHttpRequest")
        headers.setdefault("Accept", "application/json")
        url = f"{_adg_base_url()}{endpoint}"
        async with s.request(method, url, json=json_data, headers=headers) as resp:
            ct = resp.headers.get("Content-Type", "")
            # Treat empty bodies on 2xx as success
            if 200 <= resp.status < 300:
                if resp.content_length == 0:
                    return True
                if "application/json" in ct:
                    try:
                        return await resp.json()
                    except Exception:
                        return True
                txt = await resp.text()
                return True if not (txt or "").strip() else txt
            # Fallback: explicit 204 success
            if resp.status == 204:
                return True
            # Non-2xx: return None for unified error handling
            logger.error(f"ADG API Failed [{method} {endpoint}]: Status {resp.status}, Body: {await resp.text()}")
            return None
    except Exception as e:
        logger.error(f"ADG API Error [{method} {endpoint}]: {e}")
        return None

async def get_dhcp_leases() -> List[Dict]:
    # Try API
    await adg_login()
    data = await adg_api_request("GET", "/control/dhcp/status")
    leases = []
    try:
        if isinstance(data, dict):
            arr = data.get("leases") or data.get("clients") or []
            for it in arr:
                ip = it.get("ip") or it.get("IP") or it.get("Address") or ""
                name = it.get("hostname") or it.get("HostName") or it.get("Name") or ""
                mac = it.get("mac") or it.get("MAC") or ""
                if ip or name or mac:
                    leases.append({"ip": ip, "name": name or "(未知)", "mac": mac})
            if leases:
                return leases
    except Exception:
        pass
    if (getattr(Config, "ADG_LEASES_MODE", "auto") == "api"):
        return []
    # Fallback via SSH reading possible lease files
    paths = [
        "/var/lib/AdGuardHome/dhcp.leases",
        "/var/lib/adguardhome/dhcp.leases",
        "/tmp/AdGuardHome/dhcp.leases",
    ]
    for p in paths:
        content = ssh_exec(f"cat {p} 2>/dev/null")
        if content and content.strip():
            lines = content.splitlines()
            for ln in lines:
                # Try space-separated: epoch MAC IP Hostname
                parts = ln.split()
                ip, name, mac = "", "", ""
                if len(parts) >= 4:
                    mac = parts[1]
                    ip = parts[2]
                    name = parts[3]
                elif len(parts) >= 3:
                    ip = parts[0]
                    mac = parts[1]
                    name = parts[2]
                else:
                    name = ln.strip()
                leases.append({"ip": ip, "name": name or "(未知)", "mac": mac})
            if leases:
                return leases
    return []

async def wrt_adg_general(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    # Fetch all statuses in parallel
    ss_task = adg_api_request("GET", "/control/safesearch/status")
    pc_task = adg_api_request("GET", "/control/parental/status")
    sb_task = adg_api_request("GET", "/control/safebrowsing/status")
    ql_task = adg_api_request("GET", "/control/querylog/config")
    st_task = adg_api_request("GET", "/control/stats/config")
    
    ss, pc, sb, ql, st = await asyncio.gather(ss_task, pc_task, sb_task, ql_task, st_task)
    
    # Defaults
    ss_on = ss.get("enabled", False) if isinstance(ss, dict) else False
    pc_on = pc.get("enabled", False) if isinstance(pc, dict) else False
    sb_on = sb.get("enabled", False) if isinstance(sb, dict) else False
    
    ql_int = 0
    ql_on = False
    if isinstance(ql, dict):
        ql_on = ql.get("enabled", False)
        ql_int = ql.get("interval", 0) # ms
    
    st_int = 0
    st_on = False
    if isinstance(st, dict):
        st_on = st.get("enabled", False)
        st_int = st.get("interval", 0) # ms

    def fmt_dur(ms):
        if not ms: return "禁用"
        hrs = ms / 3600000
        if hrs < 24: return f"{int(hrs)}小时"
        days = hrs / 24
        return f"{int(days)}天"

    kb = [
        [InlineKeyboardButton(f"安全搜索: {'✅' if ss_on else '❌'}", callback_data=f"wrt_adg_gen_toggle_ss_{not ss_on}")],
        [InlineKeyboardButton(f"家长控制: {'✅' if pc_on else '❌'}", callback_data=f"wrt_adg_gen_toggle_pc_{not pc_on}")],
        [InlineKeyboardButton(f"浏览安全: {'✅' if sb_on else '❌'}", callback_data=f"wrt_adg_gen_toggle_sb_{not sb_on}")],
        [InlineKeyboardButton(f"查询日志: {fmt_dur(ql_int) if ql_on else '禁用'}", callback_data="wrt_adg_gen_log_cycle")],
        [InlineKeyboardButton(f"统计数据: {fmt_dur(st_int) if st_on else '禁用'}", callback_data="wrt_adg_gen_stats_cycle")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_menu")]
    ]
    await q.edit_message_text("⚙️ 通用设置", reply_markup=InlineKeyboardMarkup(kb))

async def wrt_adg_gen_toggle(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    data = q.data
    # data format: wrt_adg_gen_toggle_TYPE_BOOL
    parts = data.split("_")
    # wrt, adg, gen, toggle, type, bool
    # type: ss, pc, sb
    target = parts[4]
    val = (parts[5] == "True")
    
    ep = ""
    if target == "ss": ep = "/control/safesearch"
    elif target == "pc": ep = "/control/parental"
    elif target == "sb": ep = "/control/safebrowsing"
    
    if ep:
        await safe_callback_answer(q, "正在切换...")
        action = "enable" if val else "disable"
        # ADG requires POST body for these endpoints, empty dict is enough
        await adg_api_request("POST", f"{ep}/{action}", {})
        await wrt_adg_general(update, context)

async def wrt_adg_gen_cycle_log(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q, "切换时长...")
    # Cycle: 24h -> 7d -> 30d -> 90d -> Disable -> 24h
    # 24h = 86400000, 7d = 604800000, 30d = 2592000000, 90d = 7776000000
    steps = [86400000, 604800000, 2592000000, 7776000000, 0]
    
    curr_cfg = await adg_api_request("GET", "/control/querylog/config")
    if not isinstance(curr_cfg, dict): return
    
    curr_int = curr_cfg.get("interval", 0)
    curr_en = curr_cfg.get("enabled", False)
    if not curr_en: curr_int = 0
    
    # find next step
    next_int = steps[0]
    for i, s in enumerate(steps):
        if curr_int == s:
            next_int = steps[(i + 1) % len(steps)]
            break
        # if current is weird, default to first step
    
    curr_cfg["enabled"] = (next_int > 0)
    curr_cfg["interval"] = next_int
    await adg_api_request("POST", "/control/querylog/config", curr_cfg)
    await wrt_adg_general(update, context)

async def wrt_adg_gen_cycle_stats(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q, "切换时长...")
    steps = [86400000, 604800000, 2592000000, 7776000000, 0]
    
    curr_cfg = await adg_api_request("GET", "/control/stats/config")
    if not isinstance(curr_cfg, dict): return
    
    curr_int = curr_cfg.get("interval", 0)
    curr_en = curr_cfg.get("enabled", False)
    if not curr_en: curr_int = 0
    
    next_int = steps[0]
    for i, s in enumerate(steps):
        if curr_int == s:
            next_int = steps[(i + 1) % len(steps)]
            break
            
    curr_cfg["enabled"] = (next_int > 0)
    curr_cfg["interval"] = next_int
    await adg_api_request("POST", "/control/stats/config", curr_cfg)
    await wrt_adg_general(update, context)

async def wrt_adg_dns_advanced(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    info = await adg_api_request("GET", "/control/dns_info")
    if not isinstance(info, dict):
        await q.edit_message_text("无法获取 DNS 信息。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_dns")]]))
        return

    rl = info.get("ratelimit", 0)
    bm = info.get("blocking_mode", "default")
    dnssec = info.get("dnssec_enabled", False)
    ipv6_dis = info.get("disable_ipv6", False)
    cache = info.get("cache_size", 0) # bytes? No, usually in bytes. API says cache_size.
    
    # Blocking modes: default, null_ip, custom_ip, nxdomain
    # We can cycle them or show current.
    
    kb = [
        [InlineKeyboardButton(f"DNSSEC: {'✅' if dnssec else '❌'}", callback_data=f"wrt_adg_dns_toggle_dnssec_{not dnssec}")],
        [InlineKeyboardButton(f"禁用 IPv6: {'✅' if ipv6_dis else '❌'}", callback_data=f"wrt_adg_dns_toggle_ipv6_{not ipv6_dis}")],
        [InlineKeyboardButton(f"速率限制: {rl}/s", callback_data="wrt_adg_dns_edit_rl")],
        [InlineKeyboardButton(f"缓存大小: {int(cache/1024/1024)} MB", callback_data="wrt_adg_dns_edit_cache")],
        [InlineKeyboardButton(f"拦截模式: {bm}", callback_data="wrt_adg_dns_cycle_bm")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_dns")]
    ]
    await q.edit_message_text(f"🛠 高级 DNS 设置", reply_markup=InlineKeyboardMarkup(kb))

async def wrt_adg_dns_toggle(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    data = q.data
    parts = data.split("_")
    # wrt, adg, dns, toggle, KEY, BOOL
    key = parts[4]
    val = (parts[5] == "True")
    
    info = await adg_api_request("GET", "/control/dns_info")
    if not isinstance(info, dict): return
    
    if key == "dnssec": info["dnssec_enabled"] = val
    elif key == "ipv6": info["disable_ipv6"] = val
    
    await safe_callback_answer(q, "应用中...")
    await adg_api_request("POST", "/control/dns_config", info)
    await wrt_adg_dns_advanced(update, context)

async def wrt_adg_dns_cycle_bm(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    modes = ["default", "nxdomain", "null_ip"] # custom_ip requires IP input, skip for cycle
    
    info = await adg_api_request("GET", "/control/dns_info")
    if not isinstance(info, dict): return
    
    curr = info.get("blocking_mode", "default")
    idx = 0
    if curr in modes:
        idx = modes.index(curr)
    
    next_mode = modes[(idx + 1) % len(modes)]
    info["blocking_mode"] = next_mode
    
    await safe_callback_answer(q, f"切换为 {next_mode}...")
    await adg_api_request("POST", "/control/dns_config", info)
    await wrt_adg_dns_advanced(update, context)

async def wrt_adg_dns_edit_rl(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    _adg_wizard_states[user_id] = {"mode": "set_ratelimit"}
    await context.bot.send_message(chat_id=update.effective_chat.id, text="请输入每秒请求限制数 (0 为不限制)：", reply_markup=ForceReply(selective=True))

async def wrt_adg_dns_edit_cache(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    _adg_wizard_states[user_id] = {"mode": "set_cache_size"}
    await context.bot.send_message(chat_id=update.effective_chat.id, text="请输入缓存大小 (MB)：", reply_markup=ForceReply(selective=True))

async def wrt_adg_filters(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q, "读取过滤器列表...")
    data = await adg_api_request("GET", "/control/filtering/status")
    if not isinstance(data, dict):
        await q.edit_message_text("获取失败。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_menu")]]))
        return
    
    filters = data.get("filters", [])
    txt = "🚫 过滤器列表\n-------------------\n"
    for f in filters:
        name = f.get("name", "Unknown")
        en = f.get("enabled", False)
        cnt = f.get("rules_count", 0)
        icon = "✅" if en else "❌"
        txt += f"{icon} {name} ({cnt})\n"
        
    kb = [
        [InlineKeyboardButton("➕ 添加列表", callback_data="wrt_adg_filter_add"),
         InlineKeyboardButton("➖ 删除列表", callback_data="wrt_adg_filter_del")],
        [InlineKeyboardButton("🔄 更新所有列表", callback_data="wrt_adg_filter_refresh")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_menu")]
    ]
    await q.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(kb))

async def wrt_adg_filter_add(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    _adg_wizard_states[user_id] = {"mode": "add_filter_list"}
    await context.bot.send_message(chat_id=update.effective_chat.id, text="请输入：列表名称 列表URL (空格分隔)：", reply_markup=ForceReply(selective=True))

async def wrt_adg_filter_del(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    _adg_wizard_states[user_id] = {"mode": "del_filter_list"}
    await context.bot.send_message(chat_id=update.effective_chat.id, text="请输入要删除的列表 URL：", reply_markup=ForceReply(selective=True))

async def wrt_adg_dhcp_config(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q, "读取 DHCP 配置...")
    st = await adg_api_request("GET", "/control/dhcp/status")
    if not isinstance(st, dict):
        await q.edit_message_text("获取 DHCP 状态失败。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_dhcp")]]))
        return
        
    en = st.get("enabled", False)
    v4 = st.get("v4", {})
    gw = v4.get("gateway_ip", "-")
    start = v4.get("range_start", "-")
    end = v4.get("range_end", "-")
    mask = v4.get("subnet_mask", "-")
    
    txt = f"⚙️ DHCP 设置\n-------------------\n状态: {'✅ 启用' if en else '❌ 禁用'}\n网关: {gw}\n掩码: {mask}\n范围: {start} - {end}"
    
    kb = [
        [InlineKeyboardButton("启用服务" if not en else "禁用服务", callback_data=f"wrt_adg_dhcp_toggle_{not en}")],
        # [InlineKeyboardButton("编辑范围", callback_data="wrt_adg_dhcp_edit_range")], # Complex wizard needed, maybe later if requested
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_dhcp")]
    ]
    await q.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(kb))

async def wrt_adg_dhcp_toggle(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    val = (q.data.split("_")[-1] == "True")
    
    st = await adg_api_request("GET", "/control/dhcp/status")
    if not isinstance(st, dict): return
    
    # We must send full config back
    payload = {
        "enabled": val,
        "v4": st.get("v4"),
        "v6": st.get("v6")
    }
    await safe_callback_answer(q, "正在设置...")
    ok = await adg_api_request("POST", "/control/dhcp/set_config", payload)
    if ok:
        await wrt_adg_dhcp_config(update, context)
    else:
        await q.edit_message_text("设置失败。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_dhcp_config")]]))

async def wrt_adg_menu(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    keyboard = [
        [InlineKeyboardButton("🧾 DHCP 租约", callback_data="wrt_adg_dhcp"),
         InlineKeyboardButton("⚙️ 通用设置", callback_data="wrt_adg_general")],
        [InlineKeyboardButton("🧩 DNS 设置", callback_data="wrt_adg_dns"),
         InlineKeyboardButton("📜 规则与重写", callback_data="wrt_adg_rules")],
        [InlineKeyboardButton("🚫 过滤器", callback_data="wrt_adg_filters"),
         InlineKeyboardButton("♻️ 重启服务", callback_data="wrt_adg_restart")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_main")]
    ]
    await q.edit_message_text("🛡️ AdGuard Home 控制", reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_adg_dhcp(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q, "读取 DHCP 租约...")
    leases = await get_dhcp_leases()
    if not leases:
        kb = [
            [InlineKeyboardButton("⚙️ DHCP 设置", callback_data="wrt_adg_dhcp_config")],
            [InlineKeyboardButton("📱 使用邻居列表", callback_data="wrt_devices")],
            [InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_menu")],
        ]
        msg = "未获取到租约信息。\n"
        if getattr(Config, "ADG_LEASES_MODE", "auto") == "api":
            msg += "当前为 API 模式，仅使用接口，不读取租约文件。"
        else:
            msg += "请在 .env 设置 ADG_URL/ADG_USER/ADG_PASS 或 ADG_TOKEN，或确保租约文件可访问。"
        await q.edit_message_text(msg, reply_markup=InlineKeyboardMarkup(kb))
        return
    txt = "🧾 当前 DHCP 租约\n-------------------\n"
    for it in leases[:100]:
        ip = it.get("ip") or "?"
        name = it.get("name") or "(未知)"
        mac = it.get("mac")
        mac_str = f" [{mac}]" if mac else ""
        txt += f"• {name} ({ip}){mac_str}\n"
    
    kb = [
        [InlineKeyboardButton("⚙️ DHCP 设置", callback_data="wrt_adg_dhcp_config")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_menu")]
    ]
    await q.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(kb))

_adg_wizard_states: Dict[int, Dict] = {}

async def handle_adg_wizard(update, context: ContextTypes.DEFAULT_TYPE):
    user_id = update.effective_user.id
    text = update.message.text
    if user_id not in _adg_wizard_states:
        return False
    st = _adg_wizard_states[user_id]
    mode = st.get("mode")
    if mode == "set_upstreams":
        ups = [x.strip() for x in text.splitlines() if x.strip()]
        cfg = await adg_api_request("GET", "/control/dns_info")
        if not isinstance(cfg, dict):
            await update.message.reply_text("获取当前 DNS 配置失败。")
            return True
        cfg["upstream_dns"] = ups
        ok = await adg_api_request("POST", "/control/dns_config", cfg)
        del _adg_wizard_states[user_id]
        await update.message.reply_text("✅ 已更新上游 DNS。" if ok else "❌ 更新失败。")
        return True
    if mode == "set_bootstrap":
        boots = [x.strip() for x in text.splitlines() if x.strip()]
        cfg = await adg_api_request("GET", "/control/dns_info")
        if not isinstance(cfg, dict):
            await update.message.reply_text("获取当前 DNS 配置失败。")
            return True
        cfg["bootstrap_dns"] = boots
        ok = await adg_api_request("POST", "/control/dns_config", cfg)
        del _adg_wizard_states[user_id]
        await update.message.reply_text("✅ 已更新 Bootstrap DNS。" if ok else "❌ 更新失败。")
        return True
    if mode == "add_rewrite":
        parts = [p for p in text.split() if p.strip()]
        if len(parts) < 2:
            await update.message.reply_text("请输入：域名 与 目标，中间空格分隔。")
            return True
        payload = {"domain": parts[0], "answer": parts[1]}
        ok = await adg_api_request("POST", "/control/rewrite/add", payload)
        del _adg_wizard_states[user_id]
        await update.message.reply_text("✅ 已添加重写。" if ok else "❌ 添加失败。")
        return True
    if mode == "del_rewrite":
        parts = [p for p in text.split() if p.strip()]
        if len(parts) < 2:
            await update.message.reply_text("请输入：域名 与 目标，中间空格分隔。")
            return True
        payload = {"domain": parts[0], "answer": parts[1]}
        ok = await adg_api_request("POST", "/control/rewrite/delete", payload)
        del _adg_wizard_states[user_id]
        await update.message.reply_text("✅ 已删除重写。" if ok else "❌ 删除失败。")
        return True
    if mode in ["add_rule", "add_rule_block", "add_rule_allow", "add_rule_custom"]:
        rule_input = text.strip()
        if len(rule_input) < 3:
            await update.message.reply_text("规则太短，请重新输入。")
            return True
        final_rule = rule_input
        if mode == "add_rule_block":
            # Strip existing syntax if user typed it
            core = rule_input.replace("||", "").replace("^", "")
            final_rule = f"||{core}^"
        elif mode == "add_rule_allow":
            core = rule_input.replace("@@||", "").replace("||", "").replace("^", "")
            final_rule = f"@@||{core}^"
        
        # Read current user rules
        status = await adg_api_request("GET", "/control/filtering/status")
        lines = []
        if isinstance(status, dict):
             lines = status.get("user_rules", [])
        
        if final_rule not in lines:
            lines.append(final_rule)
            
        ok = await adg_api_request("POST", "/control/filtering/set_rules", {"rules": lines})
        await update.message.reply_text(f"✅ 已添加规则：`{final_rule}`" if ok else "❌ API 添加失败，请使用“编辑配置”粘贴到 user_rules。", parse_mode="Markdown")
        del _adg_wizard_states[user_id]
        return True
    if mode == "del_rule":
        rule_input = text.strip()
        if len(rule_input) < 3:
            await update.message.reply_text("规则太短，请重新输入。")
            return True
        
        status = await adg_api_request("GET", "/control/filtering/status")
        lines = []
        if isinstance(status, dict):
             lines = status.get("user_rules", [])
        
        # Smart delete logic: check exact match, then block syntax, then allow syntax
        targets = [rule_input, f"||{rule_input}^", f"@@||{rule_input}^"]
        deleted_count = 0
        new_lines = []
        for ln in lines:
            if ln.strip() in targets:
                deleted_count += 1
            else:
                new_lines.append(ln)
        
        if deleted_count == 0:
            await update.message.reply_text(f"未找到匹配规则：{rule_input}")
            return True

        ok = await adg_api_request("POST", "/control/filtering/set_rules", {"rules": new_lines})
        await update.message.reply_text(f"✅ 已删除 {deleted_count} 条匹配规则。" if ok else "❌ API 删除失败，请在“编辑配置”删除相应 user_rules。")
        del _adg_wizard_states[user_id]
        return True
    if mode == "set_ratelimit":
        try:
            val = int(text.strip())
            if val < 0: raise ValueError
        except ValueError:
            await update.message.reply_text("请输入有效的正整数。")
            return True
            
        cfg = await adg_api_request("GET", "/control/dns_info")
        if not isinstance(cfg, dict):
            await update.message.reply_text("获取当前 DNS 配置失败。")
            return True
        cfg["ratelimit"] = val
        ok = await adg_api_request("POST", "/control/dns_config", cfg)
        del _adg_wizard_states[user_id]
        await update.message.reply_text(f"✅ 速率限制已设置为 {val}/s。" if ok else "❌ 设置失败。")
        return True
    if mode == "set_cache_size":
        try:
            mb = int(text.strip())
            if mb < 0: raise ValueError
        except ValueError:
            await update.message.reply_text("请输入有效的正整数 (MB)。")
            return True
            
        cfg = await adg_api_request("GET", "/control/dns_info")
        if not isinstance(cfg, dict):
            await update.message.reply_text("获取当前 DNS 配置失败。")
            return True
        cfg["cache_size"] = mb * 1024 * 1024
        ok = await adg_api_request("POST", "/control/dns_config", cfg)
        del _adg_wizard_states[user_id]
        await update.message.reply_text(f"✅ 缓存大小已设置为 {mb} MB。" if ok else "❌ 设置失败。")
        return True
    if mode == "add_filter_list":
        parts = text.strip().split(maxsplit=1)
        if len(parts) < 2:
            await update.message.reply_text("格式错误，请输入：名称 URL (空格分隔)")
            return True
        name, url = parts[0], parts[1]
        payload = {"name": name, "url": url, "whitelist": False}
        ok = await adg_api_request("POST", "/control/filtering/add_url", payload)
        del _adg_wizard_states[user_id]
        if ok is True or (isinstance(ok, str) and "OK" in ok):
            await update.message.reply_text(f"✅ 已添加列表：{name}")
        else:
             # API might return text error
             err = ok if isinstance(ok, str) else "未知错误"
             await update.message.reply_text(f"❌ 添加失败：{err}")
        return True
    if mode == "del_filter_list":
        url = text.strip()
        payload = {"url": url, "whitelist": False}
        ok = await adg_api_request("POST", "/control/filtering/remove_url", payload)
        del _adg_wizard_states[user_id]
        if ok is True or (isinstance(ok, str) and "OK" in ok):
            await update.message.reply_text("✅ 已删除列表。")
        else:
             err = ok if isinstance(ok, str) else "未知错误"
             await update.message.reply_text(f"❌ 删除失败：{err}")
        return True
    return False

async def wrt_adg_restart(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q, "正在重启服务...")
    res = ssh_exec("/etc/init.d/AdGuardHome restart || /etc/init.d/adguardhome restart")
    await q.edit_message_text("✅ 已重启。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_menu")]]))

async def wrt_adg_dns(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    cfg = await adg_api_request("GET", "/control/dns_info")
    txt = "当前 DNS 配置不可用。"
    if isinstance(cfg, dict):
        ups = cfg.get("upstream_dns") or []
        boots = cfg.get("bootstrap_dns") or []
        txt = "🧩 DNS 设置\n-------------------\n上游 DNS:\n" + "\n".join([f"• {u}" for u in ups]) + "\n\nBootstrap DNS:\n" + "\n".join([f"• {b}" for b in boots])
    kb = [
        [InlineKeyboardButton("编辑上游 DNS", callback_data="wrt_adg_set_upstreams")],
        [InlineKeyboardButton("编辑 Bootstrap DNS", callback_data="wrt_adg_set_bootstrap")],
        [InlineKeyboardButton("🛠 高级设置", callback_data="wrt_adg_dns_advanced")],
        [InlineKeyboardButton("启用过滤", callback_data="wrt_adg_filter_on"),
         InlineKeyboardButton("停用过滤", callback_data="wrt_adg_filter_off")],
        [InlineKeyboardButton("刷新过滤规则", callback_data="wrt_adg_filter_refresh")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_menu")],
    ]
    await q.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(kb))

async def wrt_adg_rules(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    rewrites = await adg_api_request("GET", "/control/rewrite/list")
    rewrite_lines = []
    if isinstance(rewrites, list):
        for it in rewrites[:50]:
            d = it.get("domain")
            a = it.get("answer")
            rewrite_lines.append(f"• {d} -> {a}")
    status = await adg_api_request("GET", "/control/filtering/status")
    user_rules_count = 0
    if isinstance(status, dict):
        user_rules_count = len(status.get("user_rules", []))
    txt = "📜 规则与重写\n-------------------\n自定义规则数量: " + str(user_rules_count) + "\n\n重写记录:\n" + ("\n".join(rewrite_lines) if rewrite_lines else "无")
    kb = [
        [InlineKeyboardButton("➕ 添加规则", callback_data="wrt_adg_rule_add_menu"),
         InlineKeyboardButton("➖ 删除规则", callback_data="wrt_adg_del_rule")],
        [InlineKeyboardButton("➕ 添加重写", callback_data="wrt_adg_add_rewrite"),
         InlineKeyboardButton("➖ 删除重写", callback_data="wrt_adg_del_rewrite")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_menu")],
    ]
    await q.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(kb))

async def wrt_adg_rule_add_menu(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    kb = [
        [InlineKeyboardButton("🚫 封锁域名", callback_data="wrt_adg_add_block_start"),
         InlineKeyboardButton("✅ 放行域名", callback_data="wrt_adg_add_allow_start")],
        [InlineKeyboardButton("✏️ 自定义/Regex", callback_data="wrt_adg_add_custom_start")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_rules")]
    ]
    await q.edit_message_text("➕ 添加规则 - 请选择类型", reply_markup=InlineKeyboardMarkup(kb))

async def wrt_adg_add_block_start(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    _adg_wizard_states[user_id] = {"mode": "add_rule_block"}
    await context.bot.send_message(chat_id=update.effective_chat.id, text="🚫 请输入要封锁的域名（例如 example.com）：\n(将自动添加 ||domain^)", reply_markup=ForceReply(selective=True))

async def wrt_adg_add_allow_start(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    _adg_wizard_states[user_id] = {"mode": "add_rule_allow"}
    await context.bot.send_message(chat_id=update.effective_chat.id, text="✅ 请输入要放行的域名（例如 example.com）：\n(将自动添加 @@||domain^)", reply_markup=ForceReply(selective=True))

async def wrt_adg_add_custom_start(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    _adg_wizard_states[user_id] = {"mode": "add_rule_custom"}
    await context.bot.send_message(chat_id=update.effective_chat.id, text="✏️ 请输入自定义规则（例如 /REGEX/ 或 1.2.3.4 domain）：", reply_markup=ForceReply(selective=True))


async def wrt_adg_set_upstreams(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    _adg_wizard_states[user_id] = {"mode": "set_upstreams"}
    await context.bot.send_message(chat_id=update.effective_chat.id, text="请输入上游 DNS，每行一个：", reply_markup=ForceReply(selective=True))

async def wrt_adg_set_bootstrap(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    _adg_wizard_states[user_id] = {"mode": "set_bootstrap"}
    await context.bot.send_message(chat_id=update.effective_chat.id, text="请输入 Bootstrap DNS，每行一个：", reply_markup=ForceReply(selective=True))

async def wrt_adg_filter_on(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q, "正在启用过滤...")
    ok = await adg_api_request("POST", "/control/filtering/enable", {"enabled": True})
    await wrt_adg_dns(update, context) if ok else q.edit_message_text("启用失败。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_menu")]]))

async def wrt_adg_filter_off(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q, "正在停用过滤...")
    ok = await adg_api_request("POST", "/control/filtering/enable", {"enabled": False})
    await wrt_adg_dns(update, context) if ok else q.edit_message_text("停用失败。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_adg_menu")]]))

async def wrt_adg_filter_refresh(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q, "正在刷新过滤规则...")
    await adg_api_request("POST", "/control/filtering/refresh", {})
    await wrt_adg_dns(update, context)

async def wrt_adg_add_rewrite(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    _adg_wizard_states[user_id] = {"mode": "add_rewrite"}
    await context.bot.send_message(chat_id=update.effective_chat.id, text="请输入：域名 与 目标，中间空格分隔。", reply_markup=ForceReply(selective=True))

async def wrt_adg_del_rewrite(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    _adg_wizard_states[user_id] = {"mode": "del_rewrite"}
    await context.bot.send_message(chat_id=update.effective_chat.id, text="请输入：域名 与 目标，中间空格分隔。", reply_markup=ForceReply(selective=True))

async def adg_cleanup_test_rewrite() -> bool:
    payload = {"domain": "test.adg", "answer": "1.2.3.4"}
    ok = await adg_api_request("POST", "/control/rewrite/delete", payload)
    return bool(ok)

async def wrt_adg_add_rule(update, context: ContextTypes.DEFAULT_TYPE):
    # Backward compatibility or fallback
    return await wrt_adg_rule_add_menu(update, context)

async def wrt_adg_del_rule(update, context: ContextTypes.DEFAULT_TYPE):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    _adg_wizard_states[user_id] = {"mode": "del_rule"}
    await context.bot.send_message(chat_id=update.effective_chat.id, text="请输入要删除的规则（完整规则或域名）：", reply_markup=ForceReply(selective=True))
