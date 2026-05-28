"""Model Manager — multi-tier Qwen2 loading with hot-swap support.

Supports:
  - Qwen2-1.5B INT4 (default, ~1GB VRAM)
  - Qwen2-7B INT4 (upgrade path, ~4GB VRAM)
  - Qwen2-7B INT8 (higher quality, ~8GB VRAM)
  - Rule engine fallback (always available)

Config via env:
  QWEN_MODEL_TIER: 1.5b | 7b-int4 | 7b-int8  (default: 1.5b)
  QWEN_MODEL_DIR:   local model directory override
  QWEN_DEVICE:      auto | cpu | cuda
"""

import logging
import os
import threading
import time
from dataclasses import dataclass, field
from typing import Optional

logger = logging.getLogger(__name__)

# --- Model Tier Definitions ---

MODEL_TIERS = {
    "1.5b": {
        "model_id": "Qwen/Qwen2-1.5B-Instruct",
        "local_dir": "/models/qwen2-1.5b-int4",
        "quantization": "gptq-int4",
        "vram_gb": 1.0,
        "description": "Qwen2-1.5B INT4 — 轻量级，适合 CPU/低配 GPU",
    },
    "7b-int4": {
        "model_id": "Qwen/Qwen2-7B-Instruct",
        "local_dir": "/models/qwen2-7b-int4",
        "quantization": "gptq-int4",
        "vram_gb": 4.0,
        "description": "Qwen2-7B INT4 — 推荐生产配置，需 4GB+ VRAM",
    },
    "7b-int8": {
        "model_id": "Qwen/Qwen2-7B-Instruct",
        "local_dir": "/models/qwen2-7b-int8",
        "quantization": "bitsandbytes-8bit",
        "vram_gb": 8.0,
        "description": "Qwen2-7B INT8 — 最高质量，需 8GB+ VRAM",
    },
}


@dataclass
class ModelState:
    tier: str = "1.5b"
    loaded: bool = False
    model: object = None
    tokenizer: object = None
    load_time_s: float = 0.0
    error: Optional[str] = None


class ModelManager:
    """Manages Qwen2 model lifecycle with hot-swap support."""

    def __init__(self):
        self._lock = threading.Lock()
        self._state = ModelState()
        self._current_tier = os.environ.get("QWEN_MODEL_TIER", "1.5b")

    @property
    def loaded(self) -> bool:
        return self._state.loaded

    @property
    def current_tier(self) -> str:
        return self._current_tier

    def get_info(self) -> dict:
        """Return model status for API responses."""
        tier_info = MODEL_TIERS.get(self._current_tier, MODEL_TIERS["1.5b"])
        return {
            "loaded": self._state.loaded,
            "tier": self._current_tier,
            "model_id": tier_info["model_id"],
            "quantization": tier_info["quantization"],
            "vram_gb": tier_info["vram_gb"],
            "description": tier_info["description"],
            "load_time_s": round(self._state.load_time_s, 2),
            "error": self._state.error,
            "available_tiers": list(MODEL_TIERS.keys()),
        }

    def load(self, tier: Optional[str] = None) -> bool:
        """Load model for the given tier. Thread-safe."""
        with self._lock:
            target = tier or self._current_tier
            if target not in MODEL_TIERS:
                logger.error(f"Unknown model tier: {target}")
                return False

            # Already loaded with this tier
            if self._state.loaded and self._state.tier == target:
                return True

            # Unload current model first
            self._unload()

            info = MODEL_TIERS[target]
            model_dir = os.environ.get("QWEN_MODEL_DIR", info["local_dir"])
            model_id = info["model_id"]
            device = os.environ.get("QWEN_DEVICE", "auto")

            t0 = time.time()
            success = False

            if info["quantization"] == "gptq-int4":
                success = self._load_gptq(model_dir, model_id, device)
            elif info["quantization"] == "bitsandbytes-8bit":
                success = self._load_bnb_8bit(model_id, device)

            if not success:
                success = self._load_plain(model_id, device)

            self._state.load_time_s = time.time() - t0

            if success:
                self._state.tier = target
                self._current_tier = target
                logger.info(f"Model {target} loaded in {self._state.load_time_s:.1f}s")
            else:
                self._state.error = f"Failed to load tier {target}"
                logger.error(self._state.error)

            return success

    def switch_tier(self, tier: str) -> dict:
        """Hot-swap to a different model tier. Returns status dict."""
        if tier not in MODEL_TIERS:
            return {"success": False, "error": f"Unknown tier: {tier}"}

        success = self.load(tier)
        return {
            "success": success,
            "tier": tier,
            "info": self.get_info(),
        }

    def generate(self, prompt: str, max_new_tokens: int = 512) -> Optional[str]:
        """Generate text. Returns None if model not loaded."""
        if not self._state.loaded or self._state.model is None:
            return None

        import torch

        inputs = self._state.tokenizer(prompt, return_tensors="pt")
        if hasattr(self._state.model, "device"):
            inputs = {k: v.to(self._state.model.device) for k, v in inputs.items()}

        with torch.no_grad():
            outputs = self._state.model.generate(
                **inputs,
                max_new_tokens=max_new_tokens,
                do_sample=True,
                temperature=0.7,
                top_p=0.9,
                repetition_penalty=1.1,
            )

        new_tokens = outputs[0][inputs["input_ids"].shape[1] :]
        return self._state.tokenizer.decode(new_tokens, skip_special_tokens=True).strip()

    def _unload(self):
        """Free current model from memory."""
        if self._state.model is not None:
            del self._state.model
            del self._state.tokenizer
            self._state.model = None
            self._state.tokenizer = None

            # Force CUDA cache clear if available
            try:
                import torch
                if torch.cuda.is_available():
                    torch.cuda.empty_cache()
            except Exception:
                pass

        self._state.loaded = False
        self._state.error = None

    def _load_gptq(self, model_dir: str, model_id: str, device: str) -> bool:
        """Try loading with AutoGPTQ INT4."""
        try:
            from auto_gptq import AutoGPTQForCausalLM
            from transformers import AutoTokenizer

            src = model_dir if os.path.isdir(model_dir) else model_id
            logger.info(f"Loading {src} (GPTQ INT4)...")
            self._state.tokenizer = AutoTokenizer.from_pretrained(src, trust_remote_code=True)
            self._state.model = AutoGPTQForCausalLM.from_quantized(
                src,
                device_map=device,
                trust_remote_code=True,
                use_safetensors=True,
            )
            self._state.loaded = True
            return True
        except Exception as e:
            logger.debug(f"GPTQ load failed: {e}")
            return False

    def _load_bnb_8bit(self, model_id: str, device: str) -> bool:
        """Try loading with bitsandbytes 8-bit."""
        try:
            from transformers import AutoModelForCausalLM, AutoTokenizer, BitsAndBytesConfig

            logger.info(f"Loading {model_id} (bitsandbytes 8-bit)...")
            self._state.tokenizer = AutoTokenizer.from_pretrained(model_id, trust_remote_code=True)

            bnb_config = BitsAndBytesConfig(load_in_8bit=True)
            self._state.model = AutoModelForCausalLM.from_pretrained(
                model_id,
                quantization_config=bnb_config,
                device_map=device,
                trust_remote_code=True,
            )
            self._state.loaded = True
            return True
        except Exception as e:
            logger.debug(f"bitsandbytes 8-bit load failed: {e}")
            return False

    def _load_plain(self, model_id: str, device: str) -> bool:
        """Try loading without quantization (CPU fallback)."""
        try:
            from transformers import AutoModelForCausalLM, AutoTokenizer

            logger.info(f"Loading {model_id} (CPU, no quantization)...")
            self._state.tokenizer = AutoTokenizer.from_pretrained(model_id, trust_remote_code=True)
            self._state.model = AutoModelForCausalLM.from_pretrained(
                model_id,
                device_map="cpu",
                trust_remote_code=True,
                torch_dtype="auto",
            )
            self._state.loaded = True
            return True
        except Exception as e:
            logger.debug(f"Plain CPU load failed: {e}")
            return False


# Global singleton
manager = ModelManager()
