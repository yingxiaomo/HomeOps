import re
from telegram import InlineKeyboardButton, InlineKeyboardMarkup
from .helpers import safe_callback_answer
from .connection import ssh_exec

fw_states = {}

def parse_uci_firewall(uci_output, prefix="homeops_"):
    rules = {}
    for line in uci_output.splitlines():
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        value = value.strip("'")
        parts = key.split(".")
        if len(parts) < 2:
            continue
        section = parts[1]
        if prefix and (not section.startswith(prefix)):
            continue
        rules.setdefault(section, {})
        if len(parts) == 3:
            rules[section][parts[2]] = value
        else:
            rules[section]["_type"] = value
    return rules

async def wrt_fw_menu(update, context):
    q = update.callback_query
    await safe_callback_answer(q)
    kb = [
        [InlineKeyboardButton("🔀 端口转发列表", callback_data="wrt_fw_list_redirects"),
         InlineKeyboardButton("➕ 添加转发", callback_data="wrt_fw_add_redirect_start")],
        [InlineKeyboardButton("🛡️ 通信规则列表", callback_data="wrt_fw_list_rules"),
         InlineKeyboardButton("➕ 添加规则", callback_data="wrt_fw_add_rule_start")],
        [InlineKeyboardButton("📋 显示全部", callback_data="wrt_fw_list_all")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_main")]
    ]
    await q.edit_message_text("🔥 防火墙管理\n仅显示前缀为 homeops_ 的规则。", reply_markup=InlineKeyboardMarkup(kb))

async def wrt_fw_list_redirects(update, context):
    q = update.callback_query
    await safe_callback_answer(q, "读取配置中...")
    res = ssh_exec("uci show firewall")
    if not res:
        await q.edit_message_text("无法读取防火墙配置。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_fw_menu")]]))
        return
    rules = parse_uci_firewall(res)
    txt = "🔀 端口转发 (Redirects)\n-------------------\n"
    kb = []
    count = 0
    for sec, data in rules.items():
        if data.get("_type") != "redirect":
            continue
        count += 1
        name = sec.replace("homeops_", "")
        src_dport = data.get("src_dport", "?")
        dest_ip = data.get("dest_ip", "?")
        dest_port = data.get("dest_port", src_dport)
        proto = data.get("proto", "tcp")
        txt += f"🔹 {name}: {proto.upper()} :{src_dport} ➝ {dest_ip}:{dest_port}\n"
        kb.append([InlineKeyboardButton(f"🗑️ 删除 {name}", callback_data=f"wrt_fw_del_{sec}")])
    if count == 0:
        txt += "无记录。"
    kb.append([InlineKeyboardButton("🔙 返回", callback_data="wrt_fw_menu")])
    await q.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(kb))

async def wrt_fw_list_rules(update, context):
    q = update.callback_query
    await safe_callback_answer(q, "读取配置中...")
    res = ssh_exec("uci show firewall")
    if not res:
        await q.edit_message_text("无法读取防火墙配置。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_fw_menu")]]))
        return
    rules = parse_uci_firewall(res)
    txt = "🛡️ 通信规则 (Rules)\n-------------------\n"
    kb = []
    count = 0
    for sec, data in rules.items():
        if data.get("_type") != "rule":
            continue
        count += 1
        name = sec.replace("homeops_", "")
        src = data.get("src", "*")
        dest = data.get("dest", "*")
        dest_port = data.get("dest_port", "All")
        target = data.get("target", "?")
        txt += f"🔸 {name}: {src}➝{dest} :{dest_port} ({target})\n"
        kb.append([InlineKeyboardButton(f"🗑️ 删除 {name}", callback_data=f"wrt_fw_del_{sec}")])
    if count == 0:
        txt += "无记录。"
    kb.append([InlineKeyboardButton("🔙 返回", callback_data="wrt_fw_menu")])
    await q.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(kb))

async def wrt_fw_list_all(update, context):
    q = update.callback_query
    await safe_callback_answer(q, "读取全部规则...")
    res = ssh_exec("uci show firewall")
    if not res:
        await q.edit_message_text("无法读取防火墙配置。")
        return
    rules = parse_uci_firewall(res, prefix=None)
    txt = "📋 全部防火墙配置\n-------------------\n"
    redirects = []
    rule_lines = []
    kb = []
    for sec, data in rules.items():
        t = data.get("_type")
        tag = "HomeOps" if sec.startswith("homeops_") else "系统"
        if t == "redirect":
            name = data.get("name", sec)
            src_dport = data.get("src_dport", "?")
            dest_ip = data.get("dest_ip", "?")
            dest_port = data.get("dest_port", src_dport)
            proto = data.get("proto", "tcp")
            redirects.append(f"🔀 [{tag}] {name}: {proto.upper()} :{src_dport} ➝ {dest_ip}:{dest_port} ({sec})")
            if tag == "系统":
                kb.append([InlineKeyboardButton(f"迁移为可管理: {name}", callback_data=f"wrt_fw_rename_{sec}")])
        elif t == "rule":
            name = data.get("name", sec)
            src = data.get("src", "*")
            dest = data.get("dest", "*")
            dest_port = data.get("dest_port", "All")
            target = data.get("target", "?")
            rule_lines.append(f"🛡️ [{tag}] {name}: {src}➝{dest} :{dest_port} ({target}) ({sec})")
            if tag == "系统":
                kb.append([InlineKeyboardButton(f"迁移为可管理: {name}", callback_data=f"wrt_fw_rename_{sec}")])
    if redirects:
        txt += "Redirects:\n" + "\n".join(redirects) + "\n"
    if rule_lines:
        txt += "Rules:\n" + "\n".join(rule_lines)
    if not redirects and not rule_lines:
        txt += "无记录。"
    kb.append([InlineKeyboardButton("🔙 返回", callback_data="wrt_fw_menu")])
    await q.edit_message_text(txt, reply_markup=InlineKeyboardMarkup(kb))

async def wrt_fw_add_redirect_start(update, context):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    fw_states[user_id] = {"type": "redirect", "step": "name", "data": {}}
    await context.bot.send_message(
        chat_id=update.effective_chat.id,
        text="➕ 添加端口转发 - 第 1/5 步\n请输入规则名称 (如: web):",
        reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("取消", callback_data="wrt_fw_menu")]])
    )

async def wrt_fw_add_rule_start(update, context):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    fw_states[user_id] = {"type": "rule", "step": "name", "data": {}}
    await context.bot.send_message(
        chat_id=update.effective_chat.id,
        text="➕ 添加通信规则 - 第 1/5 步\n请输入规则名称 (如: allow_ssh):",
        reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("取消", callback_data="wrt_fw_menu")]])
    )

async def wrt_fw_wiz_finish(update, context):
    q = update.callback_query
    await safe_callback_answer(q)
    user_id = update.effective_user.id
    if user_id not in fw_states:
        await safe_callback_answer(q, "操作已过期")
        return
    state = fw_states[user_id]
    data = q.data
    if state["type"] == "redirect" and state["step"] == "proto":
        proto_map = {
            "wrt_fw_wiz_proto_tcp": "tcp",
            "wrt_fw_wiz_proto_udp": "udp",
            "wrt_fw_wiz_proto_tcpudp": "tcp udp",
        }
        state["data"]["proto"] = proto_map.get(data, "tcp")
        await wrt_fw_commit_redirect(update, context, state["data"])
        del fw_states[user_id]
    elif state["type"] == "rule" and state["step"] == "target":
        state["data"]["target"] = data.replace("wrt_fw_wiz_target_", "")
        await wrt_fw_commit_rule(update, context, state["data"])
        del fw_states[user_id]

async def wrt_fw_commit_redirect(update, context, data):
    name = data["name"]
    sec = f"homeops_{name}"
    cmds = [
        f"uci set firewall.{sec}=redirect",
        f"uci set firewall.{sec}.name='{name}'",
        f"uci set firewall.{sec}.src='wan'",
        f"uci set firewall.{sec}.src_dport='{data['ext_port']}'",
        f"uci set firewall.{sec}.dest='lan'",
        f"uci set firewall.{sec}.dest_ip='{data['int_ip']}'",
        f"uci set firewall.{sec}.dest_port='{data['int_port']}'",
        f"uci set firewall.{sec}.proto='{data['proto']}'",
        f"uci set firewall.{sec}.target='DNAT'",
        "uci commit firewall",
        "/etc/init.d/firewall reload",
    ]
    await update.callback_query.edit_message_text(f"⏳ 正在添加端口转发 {name}...")
    ssh_exec(" && ".join(cmds))
    await update.callback_query.edit_message_text("✅ 已添加。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回防火墙", callback_data="wrt_fw_menu")]]))

async def wrt_fw_commit_rule(update, context, data):
    name = data["name"]
    sec = f"homeops_{name}"
    cmds = [
        f"uci set firewall.{sec}=rule",
        f"uci set firewall.{sec}.name='{name}'",
        f"uci set firewall.{sec}.src='{data['src']}'",
        f"uci set firewall.{sec}.dest='{data['dest']}'",
    ]
    if data.get("dest_port"):
        cmds.append(f"uci set firewall.{sec}.dest_port='{data['dest_port']}'")
    cmds += [
        f"uci set firewall.{sec}.target='{data['target']}'",
        f"uci set firewall.{sec}.proto='tcp udp'",
        "uci commit firewall",
        "/etc/init.d/firewall reload",
    ]
    await update.callback_query.edit_message_text(f"⏳ 正在添加通信规则 {name}...")
    ssh_exec(" && ".join(cmds))
    await update.callback_query.edit_message_text("✅ 已添加。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回防火墙", callback_data="wrt_fw_menu")]]))

async def wrt_fw_del_confirm(update, context):
    q = update.callback_query
    await safe_callback_answer(q)
    sec = q.data.replace("wrt_fw_del_", "")
    kb = [
        [InlineKeyboardButton("✅ 确认删除", callback_data=f"wrt_fw_del_do_{sec}")],
        [InlineKeyboardButton("🔙 返回", callback_data="wrt_fw_menu")]
    ]
    await q.edit_message_text(f"确认删除: {sec}", reply_markup=InlineKeyboardMarkup(kb))

async def wrt_fw_del_do(update, context):
    q = update.callback_query
    await safe_callback_answer(q, "正在删除")
    sec = q.data.replace("wrt_fw_del_do_", "")
    ssh_exec(f"uci delete firewall.{sec} && uci commit firewall && /etc/init.d/firewall reload")
    await q.edit_message_text("✅ 已删除。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回防火墙", callback_data="wrt_fw_menu")]]))

async def wrt_fw_rename(update, context):
    q = update.callback_query
    await safe_callback_answer(q, "正在迁移为可管理...")
    sec = q.data.replace("wrt_fw_rename_", "")
    res = ssh_exec("uci show firewall")
    if not res:
        await q.edit_message_text("读取配置失败。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_fw_list_all")]]))
        return
    all_rules = parse_uci_firewall(res, prefix=None)
    if sec not in all_rules:
        await q.edit_message_text(f"未找到段: {sec}", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_fw_list_all")]]))
        return
    data = all_rules[sec]
    t = data.get("_type")
    raw_name = data.get("name", "")
    base = raw_name.lower()
    base = re.sub(r"[^a-zA-Z0-9_]", "_", base) or ("redirect" if t == "redirect" else "rule")
    idx = "0"
    m_idx = re.search(r"\[(\d+)\]", sec)
    if m_idx:
        idx = m_idx.group(1)
    new_sec = f"homeops_{base}"
    if new_sec in all_rules:
        new_sec = f"homeops_{base}_{idx}"
    cmd = f"uci rename firewall.{sec}={new_sec} && uci commit firewall && /etc/init.d/firewall reload"
    ssh_exec(cmd)
    await q.edit_message_text(f"✅ 已迁移为可管理: {new_sec}", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("📋 返回全部", callback_data="wrt_fw_list_all")]]))

async def handle_fw_wizard(update, context):
    user_id = update.effective_user.id
    if user_id not in fw_states:
        return False
    st = fw_states[user_id]
    text = update.message.text.strip()
    step = st["step"]
    if st["type"] == "redirect":
        if step == "name":
            if not re.match(r"^[a-zA-Z0-9_]+$", text):
                await update.message.reply_text("❌ 名称只能包含字母、数字和下划线。请重新输入:")
                return True
            st["data"]["name"] = text
            st["step"] = "ext_port"
            await update.message.reply_text("➕ 第 2/5 步：请输入外部端口:", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("取消", callback_data="wrt_fw_menu")]]))
        elif step == "ext_port":
            if not text.isdigit():
                await update.message.reply_text("❌ 端口必须是数字。请重新输入:")
                return True
            st["data"]["ext_port"] = text
            st["step"] = "int_ip"
            await update.message.reply_text("➕ 第 3/5 步：请输入内部 IP:", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("取消", callback_data="wrt_fw_menu")]]))
        elif step == "int_ip":
            st["data"]["int_ip"] = text
            st["step"] = "int_port"
            await update.message.reply_text("➕ 第 4/5 步：请输入内部端口:", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("取消", callback_data="wrt_fw_menu")]]))
        elif step == "int_port":
            if not text.isdigit():
                await update.message.reply_text("❌ 端口必须是数字。请重新输入:")
                return True
            st["data"]["int_port"] = text
            st["step"] = "proto"
            kb = [
                [InlineKeyboardButton("TCP", callback_data="wrt_fw_wiz_proto_tcp"),
                 InlineKeyboardButton("UDP", callback_data="wrt_fw_wiz_proto_udp")],
                [InlineKeyboardButton("TCP+UDP", callback_data="wrt_fw_wiz_proto_tcpudp")],
            ]
            await update.message.reply_text("➕ 第 5/5 步：请选择协议:", reply_markup=InlineKeyboardMarkup(kb))
    else:
        if step == "name":
            if not re.match(r"^[a-zA-Z0-9_]+$", text):
                await update.message.reply_text("❌ 名称只能包含字母、数字和下划线。请重新输入:")
                return True
            st["data"]["name"] = text
            st["step"] = "src"
            await update.message.reply_text("➕ 第 2/5 步：请输入源区域 (如 wan):", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("取消", callback_data="wrt_fw_menu")]]))
        elif step == "src":
            st["data"]["src"] = text
            st["step"] = "dest"
            await update.message.reply_text("➕ 第 3/5 步：请输入目标区域 (如 lan):", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("取消", callback_data="wrt_fw_menu")]]))
        elif step == "dest":
            st["data"]["dest"] = text
            st["step"] = "dest_port"
            await update.message.reply_text("➕ 第 4/5 步：请输入目标端口 (留空表示全部):", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("取消", callback_data="wrt_fw_menu")]]))
        elif step == "dest_port":
            st["data"]["dest_port"] = text
            st["step"] = "target"
            kb = [
                [InlineKeyboardButton("ACCEPT (允许)", callback_data="wrt_fw_wiz_target_ACCEPT"),
                 InlineKeyboardButton("DROP (丢弃)", callback_data="wrt_fw_wiz_target_DROP")],
                [InlineKeyboardButton("REJECT (拒绝)", callback_data="wrt_fw_wiz_target_REJECT")],
            ]
            await update.message.reply_text("➕ 第 5/5 步：请选择动作:", reply_markup=InlineKeyboardMarkup(kb))
    return True

