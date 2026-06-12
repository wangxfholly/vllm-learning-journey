from openai import OpenAI
import time

client = OpenAI(base_url="http://localhost:8000/v1", api_key="EMPTY")

print("=" * 60)
print("流式输出:字一个个蹦出来,并测量 TTFT(首token延迟)")
print("=" * 60)

t0 = time.time()
first_token_time = None
token_count = 0

stream = client.chat.completions.create(
    model="qwen",
    messages=[{"role": "user", "content": "请用大约150字介绍一下杭州这座城市。"}],
    temperature=0.7,
    max_tokens=256,
    stream=True,          # 关键:开启流式
)

for chunk in stream:
    delta = chunk.choices[0].delta.content
    if delta:
        if first_token_time is None:
            first_token_time = time.time() - t0   # 记录首token时刻
        token_count += 1
        print(delta, end="", flush=True)          # 实时打印,不换行

total_time = time.time() - t0
print("\n" + "=" * 60)
print(f"TTFT(首token延迟): {first_token_time:.3f}s   ← 用户多久看到第一个字")
print(f"总耗时           : {total_time:.3f}s")
print(f"输出chunk数      : {token_count}")
print(f"后续平均吐字间隔  : {(total_time - first_token_time)/max(token_count-1,1)*1000:.1f}ms/chunk (≈TPOT)")
