from vllm import LLM, SamplingParams
import os, sys, time

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "..", ".."))
target = os.path.join(REPO_ROOT, "models", "Qwen2.5-7B-Instruct-AWQ")

mode = sys.argv[1] if len(sys.argv) > 1 else "baseline"

common = dict(
    model=target,
    quantization="awq_marlin",
    gpu_memory_utilization=0.90,
    max_model_len=4096,
)

if mode == "ngram":
    print(">>> 模式: n-gram 投机解码 (无需草稿模型，从历史文本找草稿)")
    llm = LLM(
        **common,
        speculative_model="[ngram]",
        num_speculative_tokens=5,
        ngram_prompt_lookup_max=4,
    )
else:
    print(">>> 模式: 纯 7B baseline")
    llm = LLM(**common)

questions = [
    "请详细解释什么是 PagedAttention，它解决了什么问题，原理是什么。",
    "用 Python 写一个快速排序的完整实现，并加上详细注释。",
]
tok = llm.get_tokenizer()
prompts = [tok.apply_chat_template([{"role": "user", "content": q}],
           tokenize=False, add_generation_prompt=True) for q in questions]

llm.generate(prompts, SamplingParams(temperature=0, max_tokens=50), use_tqdm=False)

t0 = time.time()
outputs = llm.generate(prompts, SamplingParams(temperature=0, max_tokens=512), use_tqdm=False)
elapsed = time.time() - t0

total = sum(len(o.outputs[0].token_ids) for o in outputs)
print("=" * 60)
print(f"模式: {mode}")
print(f"总输出 tokens: {total}")
print(f"耗时: {elapsed:.2f}s")
print(f"吞吐: {total/elapsed:.2f} tokens/s")
print("=" * 60)
