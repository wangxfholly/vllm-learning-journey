from vllm import LLM, SamplingParams
from vllm.sampling_params import GuidedDecodingParams
import os, json, time

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "..", ".."))
model_path = os.path.join(REPO_ROOT, "models", "Qwen2.5-7B-Instruct-AWQ")

llm = LLM(
    model=model_path,
    quantization="awq_marlin",
    gpu_memory_utilization=0.90,
    max_model_len=2048,
)
tok = llm.get_tokenizer()

# 模拟一个智能家居 Agent 的工具调用 schema
schema = {
    "type": "object",
    "properties": {
        "intent": {"type": "string", "enum": ["control_device", "query_weather", "play_music", "unknown"]},
        "device": {"type": "string"},
        "action": {"type": "string", "enum": ["on", "off", "increase", "decrease", "none"]},
        "value": {"type": "integer"},
    },
    "required": ["intent", "device", "action", "value"],
}

user_inputs = [
    "把客厅的空调温度调高到 26 度",
    "帮我关掉卧室的灯",
    "今天天气怎么样",
]

def build_prompt(text):
    sys_msg = "你是智能家居助手，把用户指令解析为结构化 JSON。"
    return tok.apply_chat_template(
        [{"role": "system", "content": sys_msg},
         {"role": "user", "content": text}],
        tokenize=False, add_generation_prompt=True)

guided = GuidedDecodingParams(json=schema)
sp = SamplingParams(temperature=0, max_tokens=256, guided_decoding=guided)

print("=" * 60)
for text in user_inputs:
    out = llm.generate([build_prompt(text)], sp, use_tqdm=False)
    raw = out[0].outputs[0].text
    print(f"用户: {text}")
    print(f"原始输出: {raw}")
    try:
        parsed = json.loads(raw)
        print(f"json.loads 成功 ✓ -> {parsed}")
    except Exception as e:
        print(f"json.loads 失败 ✗ : {e}")
    print("-" * 60)
