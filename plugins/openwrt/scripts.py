import os
from telegram import InlineKeyboardButton, InlineKeyboardMarkup
from .helpers import safe_callback_answer
from .connection import ssh_exec

async def wrt_scripts_list(update, context):
    query = update.callback_query
    await safe_callback_answer(query, "读取脚本列表...")
    script_dir = "/root/smart"
    res = ssh_exec(f"ls {script_dir}/*.sh 2>/dev/null")
    if not res:
        await query.edit_message_text(f"目录 {script_dir} 下没有找到脚本。", reply_markup=InlineKeyboardMarkup([[InlineKeyboardButton("🔙 返回", callback_data="wrt_main")]]))
        return
    scripts = res.splitlines()
    keyboard = []
    for s in scripts:
        name = os.path.basename(s)
        keyboard.append([InlineKeyboardButton(f"▶️ {name}", callback_data=f"wrt_run_{s}")])
    keyboard.append([InlineKeyboardButton("🔙 返回", callback_data="wrt_main")])
    await query.edit_message_text(f"📂 脚本列表 ({script_dir}):\n点击即可立即运行。", reply_markup=InlineKeyboardMarkup(keyboard))

async def wrt_run_script(update, context):
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

