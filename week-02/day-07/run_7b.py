from vllm import LLM, SamplingParams
import os, time

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "..", ".."))
model_path = os.path.join(REPO_ROOT, "models", "Qwen2.5-7B-Instruct-AWQ")

print(f"Model path : {model_path}")
assert os.path.exists(os.path.join(model_path, "config.json")), "config.json not found"
print("Model files OK")

t0 = time.time()
llm = LLM(
    model=model_path,
    quantization="awq_marlin",
    gpu_memory_utilization=0.90,
    max_model_len=4096,
)
print(f"Model loaded in {time.time() - t0:.2f}s")

# 同样的问题，待会和 1.5B 对比
questions = [
    "vLLM 是什么？它和传统推理方式相比核心优势在哪？请准确回答。",
    "用 Python 写一个函数，判断一个字符串是否是回文，要求忽略大小写和非字母字符。",
]

tok = llm.get_tokenizer()
prompts = []
for q in questions:
    msgs = [{"role": "user", "content": q}]
    prompts.append(tok.apply_chat_template(msgs, tokenize=False, add_generation_prompt=True))

t0 = time.time()
outputs = llm.generate(prompts, SamplingParams(temperature=0.7, top_p=0.8, max_tokens=512))
gen_time = time.time() - t0

total_tokens = 0
for i, out in enumerate(outputs):
    text = out.outputs[0].text
    n = len(out.outputs[0].token_ids)
    total_tokens += n
    print("\n" + "=" * 60)
    print(f"Q{i+1}: {questions[i]}")
    print("-" * 60)
    print(text)

print("\n" + "=" * 60)
print(f"Total: {total_tokens} tokens in {gen_time:.2f}s = {total_tokens/gen_time:.2f} tokens/s")
