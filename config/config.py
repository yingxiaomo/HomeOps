import os
from dotenv import load_dotenv

load_dotenv()

class Config:
    BOT_TOKEN = os.getenv("BOT_TOKEN")
    ADMIN_ID = int(os.getenv("ADMIN_ID", 0))
    TG_BASE_URL = os.getenv("TG_BASE_URL", None)
    TG_PROXY = os.getenv("TG_PROXY", None)

    # Gemini
    _keys = os.getenv("GEMINI_API_KEY", "")
    GEMINI_API_KEYS = [k.strip() for k in _keys.split(",") if k.strip()]
    GEMINI_API_KEY = GEMINI_API_KEYS[0] if GEMINI_API_KEYS else None

    # OpenWrt
    OPENWRT_HOST = os.getenv("OPENWRT_HOST", "")
    OPENWRT_PORT = int(os.getenv("OPENWRT_PORT", 22))
    OPENWRT_USER = os.getenv("OPENWRT_USER", "root")
    OPENWRT_PASS = os.getenv("OPENWRT_PASS")

    # OpenClash API
    OPENCLASH_API_URL = os.getenv("OPENCLASH_API_URL", "http://127.0.0.1:9090")
    OPENCLASH_API_SECRET = os.getenv("OPENCLASH_API_SECRET", "")
    
    # AdGuard Home
    ADG_URL = os.getenv("ADG_URL", None)  # e.g., http://127.0.0.1:3000
    ADG_USER = os.getenv("ADG_USER", None)
    ADG_PASS = os.getenv("ADG_PASS", None)
    ADG_TOKEN = os.getenv("ADG_TOKEN", None)
    ADG_SSH_CONFIG_PATH = os.getenv("ADG_SSH_CONFIG_PATH", "/etc/AdGuardHome.yaml")
    ADG_LEASES_MODE = os.getenv("ADG_LEASES_MODE", "auto")  # auto|api|ssh

    if not BOT_TOKEN:
        raise ValueError("BOT_TOKEN is not set in .env file")
