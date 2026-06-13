import asyncio, aiohttp, time, json, sys, statistics

API = "http://localhost:8000/v1/chat/completions"
MODEL = "qwen7b"

# 真实业务负载:固定长system prompt(可命中prefix cache) + 变化的用户问题
SYSTEM = ("你是企业级智能客服。工作规范:" +
          "始终保持礼貌专业耐心,回答准确简洁。" * 40)
QUESTIONS = [
    "退货政策是什么?", "怎么改收货地址?", "多久发货?", "支持哪些支付?",
    "会员权益有哪些?", "如何申请发票?", "运费怎么算?", "能否加急?",
]

async def one_request(session, q):
    payload = {"model": MODEL,
               "messages": [{"role": "system", "content": SYSTEM},
                            {"role": "user", "content": q}],
               "max_tokens": 128, "temperature": 0.0, "stream": True}
    t0 = time.perf_counter(); t_first = None; n = 0
    async with session.post(API, json=payload) as resp:
        async for raw in resp.content:
            line = raw.decode("utf-8").strip()
            if not line.startswith("data:"): continue
            d = line[5:].strip()
            if d == "[DONE]": break
            try: c = json.loads(d)
            except: continue
            if c["choices"][0].get("delta", {}).get("content"):
                if t_first is None: t_first = time.perf_counter()
                n += 1
    t_end = time.perf_counter()
    if t_first is None: return None
    return {"ttft": (t_first-t0)*1000,
            "tpot": ((t_end-t_first)/max(n-1,1))*1000, "tokens": n}

def pct(v, p):
    s = sorted(v); return s[min(int(len(s)*p/100), len(s)-1)]

async def run(concurrency):
    timeout = aiohttp.ClientTimeout(total=300)
    async with aiohttp.ClientSession(timeout=timeout) as s:
        w0 = time.perf_counter()
        tasks = [one_request(s, QUESTIONS[i % len(QUESTIONS)]) for i in range(concurrency)]
        res = [r for r in await asyncio.gather(*tasks) if r]
        wall = time.perf_counter() - w0
    ttfts = [r["ttft"] for r in res]; tpots = [r["tpot"] for r in res]
    tot = sum(r["tokens"] for r in res)
    print(f"  并发{concurrency:3d}: TTFT P50={pct(ttfts,50):7.1f} P99={pct(ttfts,99):7.1f}  "
          f"TPOT P50={pct(tpots,50):5.1f}  吞吐={tot/wall:7.1f} tok/s")

async def main():
    label = sys.argv[1] if len(sys.argv) > 1 else "测试"
    print(f"\n===== {label} =====")
    for c in [1, 8, 16, 32, 48]:
        await run(c)
        await asyncio.sleep(2)

if __name__ == "__main__":
    asyncio.run(main())
