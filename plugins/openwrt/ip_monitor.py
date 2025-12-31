import os
import json
import logging
from telegram.ext import ContextTypes
from config.config import Config
from .connection import ssh_exec

logger = logging.getLogger(__name__)
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
        except Exception:
            continue
    if not v4:
        v4_res = ssh_exec("/usr/bin/curl -4 -s --max-time 5 icanhazip.com || /usr/bin/curl -4 -s --max-time 5 ifconfig.me")
        if v4_res and v4_res.strip() and " " not in v4_res:
             v4 = v4_res.strip()
    if not v6:
        v6_res = ssh_exec("/usr/bin/curl -6 -s --max-time 5 icanhazip.com || /usr/bin/curl -6 -s --max-time 5 ifconfig.co")
        if v6_res and v6_res.strip() and " " not in v6_res:
             v6 = v6_res.strip()
    if v4 and (len(v4) > 15 or "." not in v4): v4 = None
    if v6 and ":" not in v6: v6 = None
    return v4, v6

async def check_ip_job(context: ContextTypes.DEFAULT_TYPE):
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
    if application.job_queue:
        application.job_queue.run_repeating(check_ip_job, interval=60, first=10)
        logger.info("IP Monitor Job registered.")

