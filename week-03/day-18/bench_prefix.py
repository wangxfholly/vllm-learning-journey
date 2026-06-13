import asyncio, aiohttp, time, json, sys

API = "http://localhost:8000/v1/chat/completions"
MODEL = "qwen7b"

# 一个很长的共享前缀(模拟几百token的system prompt/知识背景)
LONG_PREFIX = (
    "你是一个专业的企业级智能客服助手。以下是你必须严格遵守的工作规范:"
    + "规范条款:请始终保持礼貌、专业、耐心的态度回答用户问题。" * 40
)
# 不同的短问题(前缀相同,只有这里变化)
QUESTIONS = [
    "你们的退货政策是什么?",
    "如何修改我的收货地址?",
    "订单多久能发货?",
    "支持哪些支付方式?",
    "会员有什么权益?",
]

async def one_request(session, question):
    payload = {
        "model": MODEL,
        "messages": [
            {"role": "system", "content": LONG_PREFIX},
            {"role": "user", "content": question},
        ],
        "max_tokens": 64,
        "temperature": 0.0,
        "stream": True,
    }
    t0 = time.perf_counter()
    t_first = None
    async with session.post(API, json=payload) as resp:
        async for raw in resp.content:
            line = raw.decode("utf-8").strip()
            if not line.startswith("data:"):
                continue
            data = line[len("data:"):].strip()
            if data == "[DONE]":
                break
            try:
                chunk = json.loads(data)
            except json.JSONDecodeError:
                continue
            if chunk["choices"][0].get("delta", {}).get("content"):
                if t_first is None:
                    t_first = time.perf_counter()
    ttft = (t_first - t0) * 1000 if t_first else -1
    return ttft

async def main():
    label = sys.argv[1] if len(sys.argv) > 1 else "未命名"
    timeout = aiohttp.ClientTimeout(total=120)
    async with aiohttp.ClientSession(timeout=timeout) as session:
        print(f"\n===== {label} =====")
        print(f"共享前缀长度约 {len(LONG_PREFIX)} 字符")
        # 串行发,清楚看到 第1次(冷) vs 后续(热)
        for i, q in enumerate(QUESTIONS):
            ttft = await one_request(session, q)
            tag = "冷启动(前缀未缓存)" if i == 0 else "热请求(前缀应命中)"
            print(f"  第{i+1}次 [{tag:18s}] TTFT = {ttft:7.1f} ms   Q: {q}")
            await asyncio.sleep(0.3)

if __name__ == "__main__":
    asyncio.run(main())
