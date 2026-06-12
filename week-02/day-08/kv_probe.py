from vllm import LLM
import os, sys

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "..", ".."))
model_path = os.path.join(REPO_ROOT, "models", "Qwen2.5-7B-Instruct-AWQ")

max_len = int(sys.argv[1]) if len(sys.argv) > 1 else 4096

print(f">>> Testing max_model_len = {max_len}")
llm = LLM(
    model=model_path,
    quantization="awq_marlin",
    gpu_memory_utilization=0.90,
    max_model_len=max_len,
)
print(f">>> Done for max_model_len={max_len}")
