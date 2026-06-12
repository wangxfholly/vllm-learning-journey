import asyncio, aiohttp, time, json, argparse, statistics

API = "http://localhost:8000/v1/chat/completions"
MODEL = "qwen7b"
PROMPT = "请详细解释什么是操作系统的虚拟内存机制,并举一个具体例子。"
MAX_TOKENS = 128

async def one_request(session, idx):
    """发一个流式请求,精确记录 TTFT 和每个 token 到达时间"""
    payload = {
        "model": MODEL,
        "messages": [{"role": "user", "content": PROMPT}],
        "max_tokens": MAX_TOKENS,
        "temperature": 0.0,
        "stream": True,
    }
    t_start = time.perf_counter()
    t_first = None
    n_tokens = 0
    try:
        async with session.post(API, json=payload) as resp:
            async for raw in resp.content:
                line = raw.decode("utf-8").strip()
                if not line or not line.startswith("data:"):
                    continue
                data = line[len("data:"):].strip()
                if data == "[DONE]":
                    break
                try:
                    chunk = json.loads(data)
                except json.JSONDecodeError:
                    continue
                delta = chunk["choices"][0].get("delta", {})
                if delta.get("content"):
                    now = time.perf_counter()
                    if t_first is None:
                        t_first = now          # 第一个 token 到达 -> TTFT
                    n_tokens += 1
        t_end = time.perf_counter()
        if t_first is None or n_tokens == 0:
            return None
        ttft = (t_first - t_start) * 1000              # ms
        gen_time = t_end - t_first                      # 秒
        tpot = (gen_time / max(n_tokens - 1, 1)) * 1000 # ms/token
        return {"ttft": ttft, "tpot": tpot, "tokens": n_tokens,
                "total": (t_end - t_start) * 1000}
    except Exception as e:
        print(f"  请求{idx}失败: {e}")
        return None

def pct(values, p):
    if not values:
        return 0.0
    s = sorted(values)
    k = int(len(s) * p / 100)
    return s[min(k, len(s) - 1)]

async def run_concurrency(concurrency):
    """同时发 concurrency 个请求,统计这一批的指标"""
    timeout = aiohttp.ClientTimeout(total=300)
    async with aiohttp.ClientSession(timeout=timeout) as session:
        wall_start = time.perf_counter()
        tasks = [one_request(session, i) for i in range(concurrency)]
        results = await asyncio.gather(*tasks)
        wall = time.perf_counter() - wall_start
    ok = [r for r in results if r]
    if not ok:
        print(f"并发{concurrency}: 全部失败"); return
    ttfts = [r["ttft"] for r in ok]
    tpots = [r["tpot"] for r in ok]
    total_tokens = sum(r["tokens"] for r in ok)
    throughput = total_tokens / wall   # 整个系统每秒吐多少 token
    print(f"\n===== 并发 {concurrency} (成功 {len(ok)}/{concurrency}) =====")
    print(f"  TTFT(ms)   P50={pct(ttfts,50):8.1f}  P99={pct(ttfts,99):8.1f}  avg={statistics.mean(ttfts):8.1f}")
    print(f"  TPOT(ms)   P50={pct(tpots,50):8.1f}  P99={pct(tpots,99):8.1f}  avg={statistics.mean(tpots):8.1f}")
    print(f"  吞吐       {throughput:8.1f} tok/s   (墙钟 {wall:.2f}s, 总生成 {total_tokens} tokens)")

async def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--levels", default="1,4,8,16,32")
    args = parser.parse_args()
    levels = [int(x) for x in args.levels.split(",")]
    print(f"压测目标: {API}  模型: {MODEL}  max_tokens={MAX_TOKENS}")
    print(f"并发梯度: {levels}")
    for c in levels:
        await run_concurrency(c)
        await asyncio.sleep(2)   # 每个梯度间歇,让服务回到稳态

if __name__ == "__main__":
    asyncio.run(main())
