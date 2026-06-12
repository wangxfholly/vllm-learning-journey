import asyncio, aiohttp, time

URL = "http://localhost:8000/v1/chat/completions"
QUESTION = "请用大约100字介绍一下你自己。"

async def one_request(session, idx):
    payload = {
        "model": "qwen",
        "messages": [{"role": "user", "content": QUESTION}],
        "temperature": 0.7,
        "max_tokens": 128,
    }
    t0 = time.time()
    async with session.post(URL, json=payload) as resp:
        data = await resp.json()
    latency = time.time() - t0
    out_tokens = data["usage"]["completion_tokens"]
    return latency, out_tokens

async def run_concurrency(level):
    async with aiohttp.ClientSession() as session:
        t0 = time.time()
        tasks = [one_request(session, i) for i in range(level)]
        results = await asyncio.gather(*tasks)
        wall = time.time() - t0

    latencies = [r[0] for r in results]
    total_tokens = sum(r[1] for r in results)
    print(f"\n并发数={level:>2} | 总耗时={wall:5.2f}s | "
          f"总输出={total_tokens:>4} tok | "
          f"系统吞吐={total_tokens/wall:6.1f} tok/s | "
          f"单请求平均延迟={sum(latencies)/len(latencies):.2f}s | "
          f"最慢={max(latencies):.2f}s")

async def main():
    print("="*90)
    print("并发压测:同样的问题,不同并发数,观察 [系统吞吐] 升 vs [单请求延迟] 也升")
    print("="*90)
    for level in [1, 5, 10, 20]:
        await run_concurrency(level)

asyncio.run(main())
