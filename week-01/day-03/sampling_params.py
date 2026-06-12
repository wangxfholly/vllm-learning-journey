from vllm import LLM, SamplingParams
import os

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "..", ".."))
model_path = os.path.join(REPO_ROOT, "models", "Qwen2.5-1.5B-Instruct")

llm = LLM(model=model_path, gpu_memory_utilization=0.85, max_model_len=2048)

def build(q):
    return f"<|im_start|>user\n{q}<|im_end|>\n<|im_start|>assistant\n"

def run(title, prompt, sp, n=3):
    print("\n" + "=" * 60)
    print(title)
    print("=" * 60)
    for i in range(n):
        out = llm.generate([build(prompt)], sp, use_tqdm=False)
        print(f"[第{i+1}次] {out[0].outputs[0].text.strip()}")

Q = "给一家做手冲咖啡的小店起一个有创意的名字，只回答名字。"

# ===== 实验①: temperature 对比(同一问题各跑3次) =====
run("① temperature=0.0（贪心，应该3次完全一样）", Q,
    SamplingParams(temperature=0.0, max_tokens=30))

run("① temperature=0.7（适中，略有变化）", Q,
    SamplingParams(temperature=0.7, max_tokens=30))

run("① temperature=1.5（高温，发散甚至离谱）", Q,
    SamplingParams(temperature=1.5, max_tokens=30))

# ===== 实验②: top_p 对比(高温下用 top_p 收住) =====
run("② temperature=1.5 + top_p=0.5（高温但核采样收紧）", Q,
    SamplingParams(temperature=1.5, top_p=0.5, max_tokens=30))

# ===== 实验③: 复现性(temperature=0 + seed,跨进程也一致) =====
run("③ temperature=0 + seed=42（确定性输出）", Q,
    SamplingParams(temperature=0.0, seed=42, max_tokens=30))

# ===== 实验④: repetition_penalty 治复读 =====
RQ = "请重复说三遍“我爱编程”。"
run("④ 无惩罚 repetition_penalty=1.0", RQ,
    SamplingParams(temperature=0.0, max_tokens=60))
run("④ 加惩罚 repetition_penalty=1.3", RQ,
    SamplingParams(temperature=0.0, repetition_penalty=1.3, max_tokens=60))
