import logging
import paramiko
from config.config import Config

logger = logging.getLogger(__name__)
_ssh_client = None

def _ensure_ssh():
    global _ssh_client
    if _ssh_client:
        return True
    host = Config.OPENWRT_HOST
    port = Config.OPENWRT_PORT
    user = Config.OPENWRT_USER
    pwd = Config.OPENWRT_PASS
    if not pwd:
        logger.error("SSH password not set in .env")
        return False
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    try:
        c.connect(host, port=port, username=user, password=pwd, timeout=5)
        _ssh_client = c
        return True
    except Exception as e:
        logger.error(f"SSH Error: {e}")
        _ssh_client = None
        return False

def _close_ssh():
    global _ssh_client
    try:
        if _ssh_client:
            _ssh_client.close()
    except Exception:
        pass
    _ssh_client = None

def ssh_exec(command):
    if not _ensure_ssh():
        return None
    try:
        stdin, stdout, stderr = _ssh_client.exec_command(command)
        return stdout.read().decode('utf-8')
    except Exception as e:
        logger.error(f"SSH Error: {e}")
        return None

