# Day 07 - 单卡跑 7B 大模型(量化实战)

## 做了什么
- modelscope 下载 Qwen2.5-7B-Instruct-AWQ(权重文件 5.3G,4bit)
- 用 quantization=awq_marlin 在 L4(24G)上跑起 7B
- 同题对比 7B vs 1.5B 的质量
- 抠启动日志,算清 24G 显存布局

## 关键数据(7B AWQ, gpu_util=0.90, max_model_len=4096)
- 权重显存: 5.20 GiB(FP16 要 14G,量化省 ~9G)
- 激活峰值: 1.40 GiB
- KV Cache: 13.09 GiB(占池子 60%+)
- 吞吐: 89.82 tokens/s
- Maximum concurrency for 4096 tokens: 59.84x

## 核心认知
1. 幻觉不随参数量消失。7B 把 vLLM 编成 "Vectorized LLM / 阿里云开发"(全错,
   实际是 UC Berkeley 的项目,v=virtual memory/PagedAttention),而且包装得
   比 1.5B 更专业、更难识破 → 生产必须上 RAG / 工具调用。
2. 代码/格式/抽取类任务,7B 可靠性远超 1.5B(回文函数一次写对还自带测试)。
   结论:把模型当"会推理但记性不靠谱的实习生",逻辑给它,事实喂它。
3. 显存大头是 KV Cache(60%)不是权重(24%)。量化的本质 = 把省下的显存
   换成更高的并发/吞吐。
4. 长上下文 vs 高并发抢的是同一块 KV Cache:max_model_len 翻 4 倍,并发降到 1/4。

## 下一步 Day 08
KV Cache 深挖 + 长上下文:max_model_len 怎么吃显存,长文本 OOM 怎么救。
