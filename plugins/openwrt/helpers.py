import logging

logger = logging.getLogger(__name__)

async def safe_callback_answer(query, text=None, show_alert=False):
    try:
        await query.answer(text=text, show_alert=show_alert)
    except Exception as e:
        logger.warning(f"Callback answer failed: {e}")

