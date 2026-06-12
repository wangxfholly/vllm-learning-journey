# Day 16:用 PyTorch profiler 拆开 prefill vs decode

## 方法论
- GPU 是异步的:CPU 发射 kernel 就返回,不等 GPU 算完。普通 time.time() 测的是CPU发射时间,不准。
- 必须用 torch.profiler + CUDA activity,或 torch.cuda.synchronize() 才能测准 GPU 时间。
- vLLM 0.7.3 原生支持:设环境变量 VLLM_TORCH_PROFILER_DIR(import vllm 前),用 llm.start_profile()/stop_profile() 圈住推理。
- profile 时设 enforce_eager=True 关闭 CUDA Graph,才能看到逐算子调用。
- profile 要独占干净的卡:在线服务 gpu_util=0.90 占满显存,profile 前必须先关服务。

## trace 大小已暗示结论
- prefill trace 381KB(1步) vs decode trace 57MB(200步),差150倍
- trace大小≈算子调用次数,印证 decode 是大量小kernel反复调用

## 算子 Top 榜数据
PREFILL (总114.73ms, 1步):
- marlin::Marlin 89.7%(78.8+10.9),调用112次,单次0.807ms
- flash attention 仅 1.1%

DECODE (总3773ms, 200步):
- marlin::Marlin 72.4%,调用22400次,单次0.122ms
- enable_if type internal(采样+逐token后处理)22.7%,调用200次
- flash attention 仅 0.7%

## 三个核心洞察
1. 两阶段瓶颈是同一个算子 Marlin = awq_marlin 的量化矩阵乘(QKV投影+FFN大矩阵乘),是模型"心脏算子"
2. 颠覆误解:瓶颈不是 attention(只占1%),而是矩阵乘 GEMM。FlashAttention早把注意力优化得很轻了
3. 同一Marlin两副面孔:
   - prefill单次0.807ms,一次算上千token,矩阵大又满,GPU喂饱 = compute-bound
   - decode单次0.122ms,每步只算1个新token,矩阵退化成细长向量,GPU大部分闲着,瓶颈是搬5.2GB权重 = memory-bound
   - 这就是Day15 TPOT稳(decode带宽限,加batch免费摊薄搬运)/ TTFT涨(prefill抢算力)的微观根源

## 优化方向(由profile指明)
- decode memory-bound -> 加大batch最划算(Day17 max_num_seqs),一次搬权重服务更多请求
- prefill compute-bound -> 减少重复计算(Day18 Prefix Caching,命中前缀跳过Marlin)
- speculative decoding 有用的原因:一次验证多token,把搬运成本摊给多个token
