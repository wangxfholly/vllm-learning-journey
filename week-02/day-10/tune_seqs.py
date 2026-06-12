from vllm import LLM, SamplingParams
import os, sys, time

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "..", ".."))
model_path = os.path.join(REPO_ROOT, "models", "Qwen2.5-7B-Instruct-AWQ")

max_num_seqs = int(sys.argv[1]) if len(sys.argv) > 1 else 256

print(f">>> max_num_seqs = {max_num_seqs}")
llm = LLM(
    model=model_path,
    quantization="awq_marlin",
    gpu_memory_utilization=0.90,
    max_model_len=2048,
    max_num_seqs=max_num_seqs,
)

# 构造 64 个相同结构的请求，模拟并发压力
base_q = "请写一段关于人工智能发展历史的介绍，大约 200 字。"
tok = llm.get_tokenizer()
prompt = tok.apply_chat_template([{"role": "user", "content": base_q}],
         tokenize=False, add_generation_prompt=True)
prompts = [prompt] * 64

# 预热
llm.generate(prompts[:4], SamplingParams(temperature=0, max_tokens=50), use_tqdm=False)

t0 = time.time()
outputs = llm.generate(prompts, SamplingParams(temperature=0.7, max_tokens=256), use_tqdm=False)
elapsed = time.time() - t0

total = sum(len(o.outputs[0].token_ids) for o in outputs)
print("=" * 60)
print(f"max_num_seqs   : {max_num_seqs}")
print(f"请求数         : 64")
print(f"总输出 tokens  : {total}")
print(f"总耗时         : {elapsed:.2f}s")
print(f"整体吞吐       : {total/elapsed:.2f} tokens/s")
print(f"平均每请求耗时 : {elapsed/64*max_num_seqs if max_num_seqs<=64 else elapsed:.3f}s (估算)")
print("=" * 60)
