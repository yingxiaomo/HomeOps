import os
import importlib
import logging
from telegram import Update, BotCommand
from telegram.ext import ApplicationBuilder, TypeHandler, ApplicationHandlerStop
from config.config import Config
from utils.permissions import is_whitelisted

# Configure logging
logging.basicConfig(
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    level=logging.INFO
)
logger = logging.getLogger(__name__)

async def post_init(application):
    """Set up bot commands after application initialization."""
    try:
        commands = [
            BotCommand("start", "启动机器人"),
            BotCommand("ai", "Gemini AI 助手"),
            BotCommand("clash", "OpenClash控制台"),
            BotCommand("wrt", "OpenWrt系统管理"),
            BotCommand("mail", "临时邮箱 (1secmail)"),
            BotCommand("sticker", "贴纸转图片说明"),
            BotCommand("id", "获取ID信息"),
            BotCommand("grant", "授权功能 (Admin)"),
            BotCommand("users", "用户列表 (Admin)"),
        ]
        await application.bot.set_my_commands(commands)
        logger.info("Successfully set bot commands.")
    except Exception as e:
        logger.error(f"Failed to set bot commands: {e}")

async def global_auth_handler(update: Update, context):
    """Checks if the user is authorized."""
    user = update.effective_user
    if not user:
        return # Should not happen for messages

    # Allow if Admin OR if user is in permission list
    if is_whitelisted(user.id):
        return
        
    # Block unauthorized users
    logger.warning(f"Unauthorized access attempt from user {user.id} ({user.username})")
    # Optional: await update.message.reply_text("⛔ Access Denied.")
    raise ApplicationHandlerStop

def load_plugins(application):
    plugin_dir = "plugins"
    # List all python files in the plugins directory
    for filename in os.listdir(plugin_dir):
        if filename.endswith(".py") and not filename.startswith("__"):
            module_name = filename[:-3]
            try:
                # Import the module
                module = importlib.import_module(f"{plugin_dir}.{module_name}")
                
                # Check if the module has a 'handlers' list
                if hasattr(module, "handlers"):
                    for handler in module.handlers:
                        application.add_handler(handler)
                    logger.info(f"Loaded handlers from: {module_name}")
                
                # Check if the module has an 'init_plugin' function for advanced setup (e.g., jobs)
                if hasattr(module, "init_plugin"):
                    module.init_plugin(application)
                    logger.info(f"Initialized plugin: {module_name}")

                if not hasattr(module, "handlers") and not hasattr(module, "init_plugin"):
                    logger.warning(f"Plugin {module_name} has no 'handlers' list or 'init_plugin' function.")
            
            except Exception as e:
                logger.error(f"Failed to load plugin {module_name}: {e}")

def main():
    if not Config.BOT_TOKEN:
        logger.error("No BOT_TOKEN found. Please check your .env file.")
        return

    builder = ApplicationBuilder().token(Config.BOT_TOKEN)
    
    # Apply custom API base URL if provided
    if Config.TG_BASE_URL:
        logger.info(f"Using custom API base URL: {Config.TG_BASE_URL}")
        builder.base_url(Config.TG_BASE_URL)

    # Apply proxy if configured
    if Config.TG_PROXY:
        logger.info(f"Using proxy: {Config.TG_PROXY}")
        builder.proxy(Config.TG_PROXY)
        builder.get_updates_proxy(Config.TG_PROXY)

    # Register post_init hook
    builder.post_init(post_init)

    application = builder.build()

    # Global Auth Handler (Runs first due to group=-1)
    application.add_handler(TypeHandler(Update, global_auth_handler), group=-1)

    # Load all plugins
    load_plugins(application)

    logger.info("Bot is starting...")
    application.run_polling()

if __name__ == '__main__':
    main()
