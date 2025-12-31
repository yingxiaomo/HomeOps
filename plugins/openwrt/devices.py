from telegram import InlineKeyboardButton, InlineKeyboardMarkup
from .connection import ssh_exec
from .helpers import safe_callback_answer
from .adg import get_dhcp_leases

async def wrt_devices(update, context):
    query = update.callback_query
    await safe_callback_answer(query, "获取设备列表中...")
    # Prefer ADGuard Home DHCP leases if available
    adg_leases = await get_dhcp_leases()
    if adg_leases:
        txt = "📱 当前联网设备 (ADG DHCP):\n-------------------\n"
        for it in adg_leases[:100]:
            ip = it.get("ip") or "?"
            name = it.get("name") or "(未知)"
            mac = it.get("mac")
            mac_str = f" [{mac}]" if mac else ""
            txt += f"• {name} ({ip}){mac_str}\n"
        keyboard = [[InlineKeyboardButton("🔙 返回", callback_data="wrt_main")]]
        await query.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(keyboard))
        return
    # Fallback to OpenWrt dhcp leases
    res = ssh_exec("cat /tmp/dhcp.leases")
    if not res or not res.strip():
        arp = ssh_exec("cat /proc/net/arp")
        neigh = ssh_exec("ip neigh show")
        kb = [[InlineKeyboardButton("🔙 返回", callback_data="wrt_main")]]
        if arp and arp.strip():
            txt = "📱 当前邻居列表 (ARP):\n-------------------\n"
            lines = arp.splitlines()[1:]
            count = 0
            for line in lines:
                parts = line.split()
                if len(parts) >= 4:
                    ip = parts[0]
                    mac = parts[3]
                    state = parts[5] if len(parts) > 5 else ""
                    txt += f"• {ip} [{mac}] {state}\n"
                    count += 1
            if count == 0:
                txt += "没有邻居记录。"
            await query.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(kb))
            return
        if neigh and neigh.strip():
            txt = "📱 当前邻居列表 (IP Neigh):\n-------------------\n"
            for line in neigh.splitlines():
                tokens = line.split()
                if not tokens:
                    continue
                ip = tokens[0]
                mac = None
                for i, t in enumerate(tokens):
                    if t == "lladdr" and i + 1 < len(tokens):
                        mac = tokens[i + 1]
                        break
                state = tokens[-1] if tokens else ""
                mac_str = f" [{mac}]" if mac else ""
                txt += f"• {ip}{mac_str} {state}\n"
            await query.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(kb))
            return
        await query.edit_message_text("获取失败。", reply_markup=InlineKeyboardMarkup(kb))
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
