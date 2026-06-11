from vllm import LLM, SamplingParams
import os
import time

# 脚本所在目录:  .../vllm-learning-journey/week-01/day-01/
# 模型统一放在: .../vllm-learning-journey/models/Qwen2.5-1.5B-Instruct/
# 从脚本位置往上回退两级(day-01 -> week-01 -> 仓库根),再进 models/
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "..", ".."))
model_path = os.path.join(REPO_ROOT, "models", "Qwen2.5-1.5B-Instruct")

print(f"Script dir : {SCRIPT_DIR}")
print(f"Repo root  : {REPO_ROOT}")
print(f"Model path : {model_path}")

assert os.path.exists(os.path.join(model_path, "config.json")), \
    f"config.json not found in {model_path}"
print("✓ Model files OK")

t0 = time.time()
llm = LLM(
    model=model_path,
    gpu_memory_utilization=0.85,
    max_model_len=2048,
)
print(f"✓ Model loaded in {time.time() - t0:.2f}s")

prompt = "<|im_start|>user\n用一句话解释什么是大模型？<|im_end|>\n<|im_start|>assistant\n"

t0 = time.time()
outputs = llm.generate([prompt], SamplingParams(temperature=0.7, max_tokens=200))
gen_time = time.time() - t0

print("=" * 50)
print("Generated:", outputs[0].outputs[0].text)
print("=" * 50)
print(f"Generation time: {gen_time:.2f}s")
print(f"Output tokens: {len(outputs[0].outputs[0].token_ids)}")
print(f"Throughput: {len(outputs[0].outputs[0].token_ids)/gen_time:.2f} tokens/s")
