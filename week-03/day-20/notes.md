# Day 20 — 阶段一收官:L1+L2+L3 单卡 vLLM 优化全景

## 一、今天做了什么
用 bench_final.py 模拟真实业务负载(1153 字符客服系统提示 + 8 个轮换问题),
在最终优化配置下跑了「冷启动」与「热稳态」两轮,并发 [1,8,16,32,48]。

## 二、核心数据

### 冷启动(首次请求,无缓存命中)
| 并发 | TTFT P50 | TTFT P99 | TPOT P50 | 吞吐 tok/s |
|---|---|---|---|---|
| 1  | 122.5 | 122.5 | 20.5 | 43.9   |
| 8  | 75.1  | 75.3  | 21.4 | 208.2  |
| 16 | 70.1  | 70.9  | 22.7 | 402.3  |
| 32 | 104.9 | 106.2 | 25.5 | 738.9  |
| 48 | 127.7 | 129.2 | 28.5 | 1022.3 |

### 热稳态(系统提示词 KV 命中缓存)
| 并发 | TTFT P50 | TTFT P99 | TPOT P50 | 吞吐 tok/s |
|---|---|---|---|---|
| 1  | 29.4  | 29.4  | 20.3 | 48.7   |
| 8  | 58.3  | 58.6  | 21.3 | 210.1  |
| 16 | 69.7  | 70.1  | 22.6 | 403.1  |
| 32 | 93.3  | 94.4  | 25.4 | 743.1  |
| 48 | 126.6 | 128.3 | 28.5 | 1022.4 |

## 三、三个结论(可拿去汇报)

1. **prefix caching 在真实负载下复现成功**:并发 1 的 TTFT 122.5ms → 29.4ms,
   降幅约 76%。来源是 1153 字符系统提示的 KV 命中缓存,整段跳过 Marlin 重算。

2. **冷热吞吐完全重合(1022 tok/s @ 48 并发)**:并发越高,缓存的延迟收益越被
   continuous batching 的排队时间淹没;吞吐由 GPU 算力天花板决定,缓存救不了算力。
   印证 Day 19 结论 —— 本卡短请求是「算力先到顶、显存富余」。

3. **调度健康、无长尾**:全程 P99 ≈ P50(48 并发 128.3 vs 126.6),
   无 preemption。这是 Day 17 调对 max_num_seqs + Day 19 监控验证的共同结果。

## 四、最终生产配置(阶段一定稿)

python -m vllm.entrypoints.openai.api_server \
  --model ~/vllm-learning-journey/models/Qwen2.5-7B-Instruct-AWQ \
  --quantization awq_marlin --gpu-memory-utilization 0.90 \
  --max-model-len 2048 --served-model-name qwen7b \
  --max-num-seqs 256 --enable-prefix-caching --port 8000

## 五、L1+L2+L3 全景回顾(Day 15-20)

- Day 15 出厂基线:建立 bench.py 测量基础设施(TTFT/TPOT/吞吐)
- Day 16 瓶颈定位:profiler 揪出 Marlin kernel 是「心脏算子」(prefill 89.7% / decode 72.4%),瓶颈是 GEMM 不是 attention
- Day 17 配置调优:max_num_seqs 是上限阀门不是调节旋钮,A/B 验证 B 组吞吐崩塌如期
- Day 18 针对性优化:prefix caching 命中时 TTFT 降 86%
- Day 19 内部可观测:/metrics 看 running/waiting/cache_usage/hit_rate/preemptions
- Day 20 收官:真实负载复现 prefix caching(-76%),吞吐稳定 1022 tok/s

## 六、阶段一心得
本周最大收益在 Day 18(prefix caching)。L1+L2+L3 的价值不是把数字调到多夸张,
而是把瓶颈定位清楚、把每一分收益如实归因。缓存优化延迟,不优化吞吐天花板。

下一阶段:Golang gRPC 网关,多 vLLM 实例负载均衡 + 流式 + 健康检查/failover/降级 + trace 链路监控。
