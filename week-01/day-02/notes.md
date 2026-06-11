# Day 02 — 批量推理 & 在线 Server 模式

## 目标
亲手量化 vLLM 高并发能力,跑通离线批量 + 在线 OpenAI 兼容服务。

## 实验 ① 离线批量 vs 串行(核心数据)
| 指标 | 串行 | 批量 |
|---|---|---|
| 总 tokens | 1171 | 1168 |
| 耗时 | 16.46s | 1.94s |
| 吞吐 | 71.1 tok/s | 603.1 tok/s |
| 加速比 | 1x | 8.48x |

结论:GPU 算 1 条和算 10 条耗时差不多,拼批=免费加速。这就是 continuous batching,vLLM 高并发的根本。decode 是 memory-bound,单条时算力大量空转。

## 实验 ② 在线 Server(vllm serve)
- 启动:python -m vllm.entrypoints.openai.api_server --model <path> --served-model-name qwen --port 8000
- OpenAI 兼容接口:/v1/models、/v1/chat/completions
- 返回结构与 OpenAI 完全一致:choices/message/content、usage、finish_reason

## 实验 ③ Python SDK 对接(AI Backend 关键)
- 只改 base_url="http://localhost:8000/v1" + api_key="EMPTY"
- LangChain/Eino 业务代码零改动,即可从外部 OpenAI 切到自部署模型
- 意义:模型自主可控、降本、数据不出内网

## 重要踩坑:小模型幻觉
- 问 vLLM 是什么,1.5B 模型胡编成 Vector Language Model
- 真相:vLLM 的 v=virtual memory(源自 PagedAttention),是推理引擎
- 教训:模型越小幻觉越严重;生产需更大模型或 RAG 约束;永不无条件相信小模型事实输出 → 这就是为什么需要 RAG

## 明日(Day 03)
- 关键采样参数(temperature/top_p/top_k/repetition_penalty)对输出的影响
- 或:server 并发压测,观察自动拼批
