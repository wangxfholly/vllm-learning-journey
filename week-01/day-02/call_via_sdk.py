from openai import OpenAI

# ===== 关键:只改这两行,就从"调 OpenAI"切到"调自部署 vLLM" =====
client = OpenAI(
    base_url="http://localhost:8000/v1",   # 指向本地 vLLM,不是 api.openai.com
    api_key="EMPTY",                        # 本地服务不校验,随便填
)

resp = client.chat.completions.create(
    model="qwen",                           # 对应 --served-model-name
    messages=[
        {"role": "system", "content": "你是一个简洁专业的技术助手。"},
        {"role": "user", "content": "用三点说明 vLLM 为什么适合生产部署。"},
    ],
    temperature=0.7,
    max_tokens=256,
)

print("=" * 50)
print("回答:\n", resp.choices[0].message.content)
print("=" * 50)
print("token 用量:", resp.usage)
