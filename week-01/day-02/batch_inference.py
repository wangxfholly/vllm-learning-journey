from vllm import LLM, SamplingParams
import os, time

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "..", ".."))
model_path = os.path.join(REPO_ROOT, "models", "Qwen2.5-1.5B-Instruct")

# 只加载一次模型,两个实验复用
llm = LLM(model=model_path, gpu_memory_utilization=0.85, max_model_len=2048)
sp = SamplingParams(temperature=0.7, max_tokens=128)

def build(q):
    return f"<|im_start|>user\n{q}<|im_end|>\n<|im_start|>assistant\n"

questions = [
    "用一句话解释什么是大模型？",
    "Python 和 Java 的主要区别是什么？",
    "如何提升团队协作效率？",
    "解释一下 TCP 三次握手。",
    "什么是数据库索引？",
    "微服务架构有哪些优缺点？",
    "如何理解闭包？",
    "什么是缓存穿透？",
    "Redis 为什么快？",
    "解释一下 RESTful API。",
]

def total_tokens(outs):
    return sum(len(o.outputs[0].token_ids) for o in outs)

# ===== 实验 A：逐条串行（模拟"没有批处理"）=====
print("\n" + "="*60 + "\n实验 A：逐条串行推理（10 条，一条一条来）\n" + "="*60)
t0 = time.time()
serial_outs = []
for q in questions:
    serial_outs += llm.generate([build(q)], sp)
serial_time = time.time() - t0
serial_tok = total_tokens(serial_outs)

# ===== 实验 B：一次性批量（vLLM continuous batching）=====
print("\n" + "="*60 + "\n实验 B：批量推理（10 条一次性喂进去）\n" + "="*60)
t0 = time.time()
batch_outs = llm.generate([build(q) for q in questions], sp)
batch_time = time.time() - t0
batch_tok = total_tokens(batch_outs)

# ===== 结果对比 =====
print("\n" + "="*60 + "\n吞吐对比\n" + "="*60)
print(f"串行：{serial_tok} tokens / {serial_time:.2f}s = {serial_tok/serial_time:.1f} tokens/s")
print(f"批量：{batch_tok} tokens / {batch_time:.2f}s = {batch_tok/batch_time:.1f} tokens/s")
print(f"加速比：{(batch_tok/batch_time)/(serial_tok/serial_time):.2f}x  🚀")
