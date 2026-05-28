#!/usr/bin/env bash
# Download Qwen2 models for offline deployment.
#
# Usage:
#   ./download_model.sh [tier]
#
# Tiers:
#   1.5b     — Qwen2-1.5B INT4 (default, ~1GB)
#   7b-int4  — Qwen2-7B INT4   (~4GB)
#   7b-int8  — Qwen2-7B INT8   (~8GB)
#   all      — Download all tiers

set -euo pipefail

TIER="${1:-1.5b}"
MODEL_DIR="${QWEN_MODEL_DIR:-/models}"

declare -A MODELS=(
    ["1.5b"]="Qwen/Qwen2-1.5B-Instruct"
    ["7b-int4"]="Qwen/Qwen2-7B-Instruct"
    ["7b-int8"]="Qwen/Qwen2-7B-Instruct"
)

declare -A DIRS=(
    ["1.5b"]="${MODEL_DIR}/qwen2-1.5b-int4"
    ["7b-int4"]="${MODEL_DIR}/qwen2-7b-int4"
    ["7b-int8"]="${MODEL_DIR}/qwen2-7b-int8"
)

download_tier() {
    local tier="$1"
    local model_id="${MODELS[$tier]}"
    local dest="${DIRS[$tier]}"

    echo "=== Downloading ${tier}: ${model_id} → ${dest} ==="

    mkdir -p "$dest"

    # Download using huggingface-cli (pip install huggingface_hub)
    if command -v huggingface-cli &>/dev/null; then
        huggingface-cli download "$model_id" \
            --local-dir "$dest" \
            --local-dir-use-symlinks False
    else
        echo "huggingface-cli not found, trying python..."
        python3 -c "
from huggingface_hub import snapshot_download
snapshot_download('${model_id}', local_dir='${dest}', local_dir_use_symlinks=False)
"
    fi

    echo "=== ${tier} downloaded to ${dest} ==="
}

if [[ "$TIER" == "all" ]]; then
    for t in "${!MODELS[@]}"; do
        download_tier "$t"
    done
else
    if [[ -z "${MODELS[$TIER]+x}" ]]; then
        echo "Unknown tier: $TIER"
        echo "Available: ${!MODELS[*]}"
        exit 1
    fi
    download_tier "$TIER"
fi

echo ""
echo "Done. Set QWEN_MODEL_TIER to use the model:"
echo "  export QWEN_MODEL_TIER=${TIER}"
