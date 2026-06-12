import os
# 必须在 import vllm 之前设置:告诉 vLLM profiler 把 trace 导到哪
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
TRACE_DIR = os.path.join(SCRIPT_DIR, "trace")
os.environ["VLLM_TORCH_PROFILER_DIR"] = TRACE_DIR

from vllm import LLM, SamplingParams

REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "..", ".."))
model_path = os.path.join(REPO_ROOT, "models", "Qwen2.5-7B-Instruct-AWQ")

print("=" * 60)
print("加载模型(独占整卡 profile)")
print("=" * 60)
llm = LLM(
    model=model_path,
    quantization="awq_marlin",
    gpu_memory_utilization=0.90,
    max_model_len=2048,
    enforce_eager=True,   # 关闭 CUDA Graph,让 profiler 看到真实的逐算子调用
)

# ---- 场景1:纯 PREFILL ----
# 喂一个长 prompt,只生成 1 个 token => 几乎所有时间都在 prefill
long_prompt = "请阅读以下内容并总结。" + ("人工智能技术正在快速发展。" * 80)
sp_prefill = SamplingParams(temperature=0, max_tokens=1)

# ---- 场景2:纯 DECODE ----
# 短 prompt,生成很多 token => prefill 极短,时间几乎都在 decode
short_prompt = "讲一个故事"
sp_decode = SamplingParams(temperature=0, max_tokens=200)

# 先各跑一次预热(触发 CUDA 初始化/编译,不计入 profile)
print("\n预热中...")
llm.generate([short_prompt], SamplingParams(temperature=0, max_tokens=5))

# ===== Profile 场景1: PREFILL =====
print("\n" + "=" * 60)
print("Profiling 场景1: 纯 PREFILL (长prompt + 只生成1 token)")
print("=" * 60)
llm.start_profile()
llm.generate([long_prompt], sp_prefill)
llm.stop_profile()

# ===== Profile 场景2: DECODE =====
print("\n" + "=" * 60)
print("Profiling 场景2: 纯 DECODE (短prompt + 生成200 token)")
print("=" * 60)
llm.start_profile()
llm.generate([short_prompt], sp_decode)
llm.stop_profile()

print("\n完成。trace 已导出到:", TRACE_DIR)
print("trace 文件列表:")
for f in sorted(os.listdir(TRACE_DIR)):
    full = os.path.join(TRACE_DIR, f)
    print(f"  {f}  ({os.path.getsize(full)/1024:.0f} KB)")
