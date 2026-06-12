import os, gzip, json, glob
from collections import defaultdict

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
TRACE_DIR = os.path.join(SCRIPT_DIR, "trace")

def load_trace(path):
    with gzip.open(path, "rt", encoding="utf-8") as f:
        return json.load(f)

def analyze(path, title):
    data = load_trace(path)
    events = data.get("traceEvents", [])
    # 只看在 GPU 上真正执行的 kernel:cat == "kernel"
    kernel_time = defaultdict(float)
    kernel_count = defaultdict(int)
    total_gpu = 0.0
    for e in events:
        if e.get("cat") == "kernel" and "dur" in e:
            name = e["name"]
            dur = e["dur"]  # 微秒
            kernel_time[name] += dur
            kernel_count[name] += 1
            total_gpu += dur
    print("\n" + "=" * 70)
    print(f"  {title}")
    print(f"  GPU kernel 总耗时: {total_gpu/1000:.2f} ms   (kernel 种类 {len(kernel_time)})")
    print("=" * 70)
    print(f"{'占比':>6} {'总耗时(ms)':>11} {'调用次数':>8}  算子(kernel)")
    print("-" * 70)
    ranked = sorted(kernel_time.items(), key=lambda x: x[1], reverse=True)
    for name, t in ranked[:10]:
        pct = t / total_gpu * 100 if total_gpu else 0
        short = name if len(name) <= 48 else name[:45] + "..."
        print(f"{pct:5.1f}% {t/1000:11.3f} {kernel_count[name]:8d}  {short}")

files = sorted(glob.glob(os.path.join(TRACE_DIR, "*.pt.trace.json.gz")),
               key=os.path.getsize)
# 文件小的是 prefill(1步),大的是 decode(200步)
analyze(files[0], "场景1: 纯 PREFILL (长prompt, 生成1 token)")
analyze(files[1], "场景2: 纯 DECODE (短prompt, 生成200 token)")
