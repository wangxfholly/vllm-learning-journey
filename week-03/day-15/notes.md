# Day 15:建立性能基线 (baseline)

## 核心原则
先测量,再优化。没有 baseline,所有"提升了多少"都是玄学。

## 工具
vLLM 0.7.3 的 pip wheel 不带 benchmark_serving.py,自己写了 bench.py:
- asyncio + aiohttp 并发压测
- 必须用 stream=True 才能测出 TTFT
- TTFT = 第一个 token 到达时刻 - 发请求时刻
- TPOT = (最后token时刻 - 第一个token时刻) / (token数 - 1)
- 固定 prompt + max_tokens=128,变量越少基线越干净

## 出厂基线数据 (Qwen2.5-7B-AWQ, max_model_len=2048)
并发1:  TTFT 33.4ms   TPOT 20.2ms   吞吐 49.2 tok/s
并发4:  TTFT 60.9ms   TPOT 20.6ms   吞吐 191.6 tok/s
并发8:  TTFT 123.6ms  TPOT 21.1ms   吞吐 365.6 tok/s
并发16: TTFT 237.9ms  TPOT 22.1ms   吞吐 672.4 tok/s
并发32: TTFT 475.3ms  TPOT 24.7ms   吞吐 1129.7 tok/s

## 三个关键结论
1. 吞吐随并发涨23倍(1->32):continuous batching 把多请求在GPU上动态拼批
2. TPOT几乎不变(20.2->24.7,仅+22%):decode是memory-bound,权重读取成本被batch摊薄,batch越大越划算
3. TTFT线性恶化(33->475,14倍):prefill是compute-bound,算力被瓜分+排队,这是高并发的代价

## 作战坐标
decode便宜(TPOT稳)、prefill贵(TTFT涨)。
核心矛盾:如何在不爆TTFT的前提下,把并发/吞吐推更高。
- 32并发吞吐还在涨没饱和 -> 卡还有余量,Day17往64/128推找天花板
- Day18 Prefix Caching 直接打 TTFT
- Golang网关按队列深度路由,避免TTFT爆炸
