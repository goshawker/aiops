"""Qwen2 Multi-tier LLM Engine.

Phase 2: Qwen2-1.5B INT4 with rule engine fallback.
Phase 3: Upgraded to support Qwen2-7B via ModelManager.

Model tiers:
  - 1.5b:     Qwen2-1.5B INT4 (~1GB, default)
  - 7b-int4:  Qwen2-7B INT4  (~4GB, recommended production)
  - 7b-int8:  Qwen2-7B INT8  (~8GB, highest quality)

Config via env:
  QWEN_MODEL_TIER: 1.5b | 7b-int4 | 7b-int8  (default: 1.5b)
"""

import logging
import os
from typing import Optional

from ai.llm.model_manager import manager

logger = logging.getLogger(__name__)

MAX_NEW_TOKENS = int(os.environ.get("QWEN_MAX_TOKENS", "512"))

# --- Prompts ---

HEALTH_SUMMARY_PROMPT = """你是一个运维专家。根据以下系统状态数据，生成一段简洁的中文健康摘要。

系统指标：
{metrics}

日志摘要：
{logs}

活跃事件：
{incidents}

请用一段话总结系统状态，指出问题并给出建议。不超过200字。"""

CHAT_SYSTEM_PROMPT = """你是一个 AIOps 智能运维助手。你可以：
1. 分析系统指标和日志
2. 解释告警原因
3. 提供运维建议
4. 回答技术问题

请用简洁专业的中文回答。"""


def _ensure_loaded():
    """Ensure model is loaded (lazy init on first call)."""
    if not manager.loaded:
        manager.load()


def generate_summary(
    metrics_text: str,
    logs_text: str,
    incidents_text: str,
) -> Optional[str]:
    """Generate health summary using Qwen2 model.

    Returns None if model is not available.
    """
    _ensure_loaded()
    if not manager.loaded:
        return None

    prompt = HEALTH_SUMMARY_PROMPT.format(
        metrics=metrics_text,
        logs=logs_text,
        incidents=incidents_text,
    )

    try:
        return manager.generate(prompt, max_new_tokens=MAX_NEW_TOKENS)
    except Exception as e:
        logger.error(f"Summary generation failed: {e}")
        return None


def chat(message: str, history: list[dict] = None) -> Optional[str]:
    """Chat with the Qwen2 model.

    Returns None if model is not available.
    """
    _ensure_loaded()
    if not manager.loaded:
        return None

    # Build chat prompt
    prompt = CHAT_SYSTEM_PROMPT + "\n\n"
    if history:
        for h in history[-5:]:  # Last 5 turns
            role = h.get("role", "user")
            content = h.get("content", "")
            if role == "user":
                prompt += f"用户：{content}\n"
            else:
                prompt += f"助手：{content}\n"
    prompt += f"用户：{message}\n助手："

    try:
        return manager.generate(prompt, max_new_tokens=MAX_NEW_TOKENS)
    except Exception as e:
        logger.error(f"Chat generation failed: {e}")
        return None


def get_model_info() -> dict:
    """Return model status info."""
    return manager.get_info()


def switch_model_tier(tier: str) -> dict:
    """Switch to a different model tier. Returns status dict."""
    return manager.switch_tier(tier)
