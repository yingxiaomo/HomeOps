import json
import os
import logging
from config.config import Config

PERM_FILE = "data/permissions.json"
logger = logging.getLogger(__name__)

if not os.path.exists("data"):
    os.makedirs("data")

def load_permissions():
    if not os.path.exists(PERM_FILE):
        return {}
    try:
        with open(PERM_FILE, 'r') as f:
            return json.load(f)
    except Exception as e:
        logger.error(f"Failed to load permissions: {e}")
        return {}

def save_permissions(perms):
    try:
        with open(PERM_FILE, 'w') as f:
            json.dump(perms, f, indent=4)
    except Exception as e:
        logger.error(f"Failed to save permissions: {e}")

def grant_permission(user_id, feature):
    perms = load_permissions()
    str_id = str(user_id)
    
    if str_id not in perms:
        perms[str_id] = []
    
    if feature not in perms[str_id]:
        perms[str_id].append(feature)
        save_permissions(perms)
        return True
    return False

def revoke_permission(user_id, feature):
    perms = load_permissions()
    str_id = str(user_id)
    
    if str_id in perms and feature in perms[str_id]:
        perms[str_id].remove(feature)
        if not perms[str_id]:
            del perms[str_id]
        save_permissions(perms)
        return True
    return False

def has_permission(user_id, feature):
    if user_id == Config.ADMIN_ID:
        return True
        
    perms = load_permissions()
    str_id = str(user_id)
    return str_id in perms and feature in perms[str_id]

def is_admin(user_id):
    return user_id == Config.ADMIN_ID

def is_whitelisted(user_id):
    """Check if user is admin OR has ANY permission."""
    if is_admin(user_id):
        return True
    perms = load_permissions()
    return str(user_id) in perms
