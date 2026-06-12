from vllm import LLM, SamplingParams
import os, sys, time, torch

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "..", ".."))

# 通过命令行参数决定跑哪个模型: fp16 或 awq
which = sys.argv[1] if len(sys.argv) > 1 else "fp16"
if which == "fp16":
    model_dir = "Qwen2.5-1.5B-Instruct"
    tag = "FP16 原版"
elif which == "awq":
    model_dir = "Qwen2.5-1.5B-Instruct-AWQ"
    tag = "AWQ 4bit 量化版"
else:
    print("用法: python quant_compare.py [fp16|awq]"); sys.exit(1)

model_path = os.path.join(REPO_ROOT, "models", model_dir)

print("\n" + "#" * 60)
print(f"# 正在测试: {tag}")
print("#" * 60)

llm = LLM(model=model_path, gpu_memory_utilization=0.85, max_model_len=2048)
sp = SamplingParams(temperature=0.0, max_tokens=128)  # 贪心,保证可对比

def build(q):
    return f"<|im_start|>user\n{q}<|im_end|>\n<|im_start|>assistant\n"

# ---- 测显存:权重实际占用 ----
mem_alloc = torch.cuda.memory_allocated() / (1024**3)
mem_reserved = torch.cuda.memory_reserved() / (1024**3)

# ---- 测速度:跑同一问题 ----
Q = "请详细解释什么是机器学习中的过拟合，以及如何避免。"
# warmup
llm.generate([build("你好")], sp, use_tqdm=False)

t0 = time.time()
out = llm.generate([build(Q)], sp, use_tqdm=False)
dt = time.time() - t0
text = out[0].outputs[0].text.strip()
n_tok = len(out[0].outputs[0].token_ids)

print("\n" + "=" * 60)
print(f"【{tag}】结果")
print("=" * 60)
print(f"权重显存(allocated): {mem_alloc:.2f} GB")
print(f"速度: {n_tok} tokens / {dt:.2f}s = {n_tok/dt:.1f} tokens/s")
print(f"输出质量:\n{text}")
print("=" * 60)
