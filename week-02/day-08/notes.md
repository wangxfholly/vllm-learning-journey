# Day 08 - KV Cache 深挖 + 长上下文的代价

## 核心实验:max_model_len 对并发的影响(7B AWQ, gpu_util=0.90)
| max_model_len | 激活峰值 | KV Cache | cuda blocks | 并发数 |
|---|---|---|---|---|
| 2048  | 1.40 GiB | 13.10 GiB | 15336 | 119.81x |
| 8192  | 中间     | 13.06 GiB | 15289 | 29.86x  |
| 32768 | 5.51 GiB | 8.99 GiB  | 10523 | 5.14x   |

## 铁律
并发数 = KV缓存池容量 / 单请求长度
- 缓存池 = cuda blocks × 16 token/block
- 验证: 8192 → 245376/8192=29.9(实测29.86); 32768 → 168368/32768=5.14(实测5.14)

## 最深刻的一课:长上下文是"双重打击"
长度↑ 同时造成:
1. 每请求 KV Cache 占用↑(线性，已知)
2. PyTorch 激活峰值↑(隐藏！2048→32768 时激活从 1.40G 飙到 5.51G)
   → 抢走近 4G，KV Cache 池从 13.10G 缩到 8.99G
结果: 并发从 120 跌到 5(24倍暴跌)，远超简单 16 倍反比。
结论: 生产里 max_model_len 必须按实际需要设，无脑设满代价极大。

## KV Cache 原理(PagedAttention)
- 按 block 管理(默认 16 token/block)，借鉴 OS 分页内存，几乎无碎片
- # cuda blocks=GPU 缓存块; # CPU blocks=swap 兜底(换出到内存)
- CUDA Graph: 录制 decode 流重放省 CPU 调度，代价 0.26G 显存

## 长文本 OOM 三种救法
1. 调大 gpu_memory_utilization(扩缓存池，暴力)
2. 调小 max_model_len(降单请求开销，最该优先)
3. enforce_eager=True(关 CUDA Graph 省显存，代价 decode 变慢)
进阶: kv_cache_dtype=fp8(KV量化容量翻倍); CPU swap

## 下一步 Day 09
投机解码 speculative decoding: 小模型当草稿，大模型当审稿，白嫖加速。
