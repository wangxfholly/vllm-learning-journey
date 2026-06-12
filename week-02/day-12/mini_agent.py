"""
Mini Agent - 两周 vLLM 学习的综合项目
本地 7B 模型 + guided decoding 工具调用 + 流式回复，零外部 API。
"""
from vllm import LLM, SamplingParams
from vllm.sampling_params import GuidedDecodingParams
import os, json

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "..", ".."))
model_path = os.path.join(REPO_ROOT, "models", "Qwen2.5-7B-Instruct-AWQ")

# ============ 1. 定义工具（真实可执行的 Python 函数）============
def get_weather(city: str) -> str:
    fake_db = {"北京": "晴 24°C", "上海": "多云 27°C", "深圳": "雷阵雨 30°C"}
    return fake_db.get(city, f"{city} 暂无天气数据")

def calculate(expression: str) -> str:
    try:
        # 仅允许数字和基本运算符，安全求值
        allowed = set("0123456789+-*/(). ")
        if not set(expression) <= allowed:
            return "表达式含非法字符"
        return str(eval(expression))
    except Exception as e:
        return f"计算错误: {e}"

TOOLS = {"get_weather": get_weather, "calculate": calculate}

# ============ 2. 工具选择的 schema（guided decoding 锁死）============
router_schema = {
    "type": "object",
    "properties": {
        "tool": {"type": "string", "enum": ["get_weather", "calculate", "none"]},
        "city": {"type": "string"},
        "expression": {"type": "string"},
    },
    "required": ["tool", "city", "expression"],
}

# ============ 3. 加载模型（Day 07/08 的配置）============
print(">>> 加载 7B 模型中...")
llm = LLM(
    model=model_path,
    quantization="awq_marlin",
    gpu_memory_utilization=0.90,
    max_model_len=2048,
)
tok = llm.get_tokenizer()
print(">>> 模型就绪\n")

def chat(prompt_text, system, sampling):
    msgs = [{"role": "system", "content": system},
            {"role": "user", "content": prompt_text}]
    text = tok.apply_chat_template(msgs, tokenize=False, add_generation_prompt=True)
    return llm.generate([text], sampling, use_tqdm=False)[0].outputs[0].text

def run_agent(user_input):
    print(f"\n{'='*60}\n用户: {user_input}")

    # --- Step 1: 工具路由（temperature=0 求稳，guided 锁格式）---
    router_sys = ("你是工具路由器。根据用户输入选择工具并填参数。"
                  "查天气用 get_weather 填 city；计算用 calculate 填 expression(纯数学表达式)；"
                  "都不是用 none。不需要的字段填空字符串。")
    guided = GuidedDecodingParams(json=router_schema)
    route_raw = chat(user_input, router_sys,
                     SamplingParams(temperature=0, max_tokens=128, guided_decoding=guided))
    route = json.loads(route_raw)
    print(f"[路由决策] {route}")

    # --- Step 2: 执行工具 ---
    tool = route["tool"]
    if tool == "get_weather":
        result = get_weather(route["city"])
    elif tool == "calculate":
        result = calculate(route["expression"])
    else:
        result = None
    print(f"[工具结果] {result}")

    # --- Step 3: 生成自然语言回复（temperature=0.7 求自然）---
    if result is not None:
        reply_sys = "你是友好的助手。根据工具返回的结果，用自然的中文回答用户。"
        reply_input = f"用户问：{user_input}\n工具返回：{result}\n请回复用户。"
    else:
        reply_sys = "你是友好的助手，直接回答用户问题。"
        reply_input = user_input
    reply = chat(reply_input, reply_sys, SamplingParams(temperature=0.7, max_tokens=256))
    print(f"[最终回复] {reply.strip()}")

# ============ 4. 跑几个测试 case ============
for q in ["北京今天天气怎么样？", "帮我算一下 (135 + 27) * 8 等于多少", "你叫什么名字？"]:
    run_agent(q)
