from telegram.ext import ContextTypes, CommandHandler, CallbackQueryHandler, MessageHandler, filters
from .menu import wrt_menu
from .menu import wrt_exit
from .status import wrt_status, wrt_show_current_ips, wrt_reboot_confirm, wrt_reboot_do, wrt_services_menu, wrt_svc_action, wrt_drop_caches
from .net import wrt_net_menu, wrt_net_ping_ask, wrt_net_trace_ask, wrt_net_nslookup_ask, wrt_net_curl_ask, wrt_net_myip, handle_wrt_message, wrt_net_quick, wrt_net_manual, wrt_net_run_quick
from .devices import wrt_devices
from .firewall import wrt_fw_menu, wrt_fw_list_redirects, wrt_fw_list_rules, wrt_fw_list_all, wrt_fw_rename, wrt_fw_add_redirect_start, wrt_fw_add_rule_start, wrt_fw_wiz_finish, wrt_fw_del_confirm, wrt_fw_del_do
from .scripts import wrt_scripts_list, wrt_run_script
from .adg import (
    wrt_adg_menu, wrt_adg_dhcp, wrt_adg_restart,
    wrt_adg_dns, wrt_adg_rules, wrt_adg_set_upstreams, wrt_adg_set_bootstrap,
    wrt_adg_filter_on, wrt_adg_filter_off, wrt_adg_filter_refresh,
    wrt_adg_add_rewrite, wrt_adg_del_rewrite, wrt_adg_add_rule, wrt_adg_del_rule,
    wrt_adg_rule_add_menu, wrt_adg_add_block_start, wrt_adg_add_allow_start, wrt_adg_add_custom_start,
    wrt_adg_general, wrt_adg_gen_toggle, wrt_adg_gen_cycle_log, wrt_adg_gen_cycle_stats,
    wrt_adg_dns_advanced, wrt_adg_dns_toggle, wrt_adg_dns_cycle_bm, wrt_adg_dns_edit_rl, wrt_adg_dns_edit_cache,
    wrt_adg_filters, wrt_adg_filter_add, wrt_adg_filter_del,
    wrt_adg_dhcp_config, wrt_adg_dhcp_toggle
)

async def handle_callback(update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    data = query.data
    if data == "wrt_main":
        await wrt_menu(update, context)
    elif data == "wrt_exit":
        await wrt_exit(update, context)
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
    elif data == "wrt_net_quick":
        await wrt_net_quick(update, context)
    elif data == "wrt_net_manual":
        await wrt_net_manual(update, context)
    elif data.startswith("wrt_net_run_"):
        await wrt_net_run_quick(update, context)
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
    elif data == "wrt_services_menu":
        await wrt_services_menu(update, context)
    elif data.startswith("wrt_svc_restart_"):
        await wrt_svc_action(update, context)
    elif data == "wrt_drop_caches":
        await wrt_drop_caches(update, context)
    elif data == "wrt_fw_menu":
        await wrt_fw_menu(update, context)
    elif data == "wrt_fw_list_redirects":
        await wrt_fw_list_redirects(update, context)
    elif data == "wrt_fw_list_rules":
        await wrt_fw_list_rules(update, context)
    elif data == "wrt_fw_add_redirect_start":
        await wrt_fw_add_redirect_start(update, context)
    elif data == "wrt_fw_add_rule_start":
        await wrt_fw_add_rule_start(update, context)
    elif data == "wrt_fw_list_all":
        await wrt_fw_list_all(update, context)
    elif data.startswith("wrt_fw_rename_"):
        await wrt_fw_rename(update, context)
    elif data.startswith("wrt_fw_wiz_"):
        await wrt_fw_wiz_finish(update, context)
    elif data.startswith("wrt_fw_del_"):
        await wrt_fw_del_confirm(update, context)
    elif data.startswith("wrt_fw_del_do_"):
        await wrt_fw_del_do(update, context)
    elif data == "wrt_scripts_list":
        await wrt_scripts_list(update, context)
    elif data.startswith("wrt_run_"):
        await wrt_run_script(update, context)
    elif data == "wrt_adg_menu":
        await wrt_adg_menu(update, context)
    elif data == "wrt_adg_dhcp":
        await wrt_adg_dhcp(update, context)
    elif data == "wrt_adg_dns":
        await wrt_adg_dns(update, context)
    elif data == "wrt_adg_rules":
        await wrt_adg_rules(update, context)
    elif data == "wrt_adg_set_upstreams":
        await wrt_adg_set_upstreams(update, context)
    elif data == "wrt_adg_set_bootstrap":
        await wrt_adg_set_bootstrap(update, context)
    elif data == "wrt_adg_filter_on":
        await wrt_adg_filter_on(update, context)
    elif data == "wrt_adg_filter_off":
        await wrt_adg_filter_off(update, context)
    elif data == "wrt_adg_filter_refresh":
        await wrt_adg_filter_refresh(update, context)
    elif data == "wrt_adg_restart":
        await wrt_adg_restart(update, context)
    elif data == "wrt_adg_add_rewrite":
        await wrt_adg_add_rewrite(update, context)
    elif data == "wrt_adg_del_rewrite":
        await wrt_adg_del_rewrite(update, context)
    elif data == "wrt_adg_add_rule":
        await wrt_adg_add_rule(update, context)
    elif data == "wrt_adg_del_rule":
        await wrt_adg_del_rule(update, context)
    elif data == "wrt_adg_rule_add_menu":
        await wrt_adg_rule_add_menu(update, context)
    elif data == "wrt_adg_add_block_start":
        await wrt_adg_add_block_start(update, context)
    elif data == "wrt_adg_add_allow_start":
        await wrt_adg_add_allow_start(update, context)
    elif data == "wrt_adg_add_custom_start":
        await wrt_adg_add_custom_start(update, context)
    elif data == "wrt_adg_general":
        await wrt_adg_general(update, context)
    elif data.startswith("wrt_adg_gen_toggle_"):
        await wrt_adg_gen_toggle(update, context)
    elif data == "wrt_adg_gen_log_cycle":
        await wrt_adg_gen_cycle_log(update, context)
    elif data == "wrt_adg_gen_stats_cycle":
        await wrt_adg_gen_cycle_stats(update, context)
    elif data == "wrt_adg_dns_advanced":
        await wrt_adg_dns_advanced(update, context)
    elif data.startswith("wrt_adg_dns_toggle_"):
        await wrt_adg_dns_toggle(update, context)
    elif data == "wrt_adg_dns_cycle_bm":
        await wrt_adg_dns_cycle_bm(update, context)
    elif data == "wrt_adg_dns_edit_rl":
        await wrt_adg_dns_edit_rl(update, context)
    elif data == "wrt_adg_dns_edit_cache":
        await wrt_adg_dns_edit_cache(update, context)
    elif data == "wrt_adg_filters":
        await wrt_adg_filters(update, context)
    elif data == "wrt_adg_filter_add":
        await wrt_adg_filter_add(update, context)
    elif data == "wrt_adg_filter_del":
        await wrt_adg_filter_del(update, context)
    elif data == "wrt_adg_dhcp_config":
        await wrt_adg_dhcp_config(update, context)
    elif data.startswith("wrt_adg_dhcp_toggle_"):
        await wrt_adg_dhcp_toggle(update, context)

handlers = [
    CommandHandler("wrt", wrt_menu),
    CallbackQueryHandler(handle_callback, pattern="^wrt_"),
    MessageHandler(filters.TEXT & (~filters.COMMAND), handle_wrt_message)
]
