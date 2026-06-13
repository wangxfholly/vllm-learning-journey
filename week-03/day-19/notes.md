# Day 19:Prometheus 指标与可观测性

## 两个视角看同一系统
- 外部黑盒(bench.py):用户体感的 TTFT/TPOT/吞吐
- 内部白盒(/metrics):系统自报的 running/waiting/cache/命中率
生产排障要两者结合

## /metrics 三类指标
1. 配置信息(静态): cache_config_info 暴露 block_size=16, num_gpu_blocks=15336,
   enable_prefix_caching 等启动配置。15336*16/2048≈119.81x 对上Day08的concurrency
2. 实时状态(动态,监控核心):
   - num_requests_running: 当前在GPU跑的请求数 = batch大小
   - num_requests_waiting: 排队数,持续>0=过载该扩容(告警关键)
   - num_requests_swapped: 被换出CPU,>0=显存吃紧
   - gpu_cache_usage_perc: KV用了百分之几,接近100%要爆
3. 累计计数(单调增,算速率): prompt/generation_tokens_total,
   num_preemptions_total, gpu_prefix_cache_hit_rate
   Prometheus用rate()算变化速率,如rate(generation_tokens_total[1m])=实时吞吐

## 实测:压测48并发时抓到的状态
- num_requests_running=48 精准对应压测并发(内部能看到batch大小了)
- num_requests_waiting=0 证明48并发这台卡很从容(对比Day17 B组会是running=32 waiting=16)
- gpu_cache_usage_perc=0.35% 极低! -> 揭示真相:短请求场景算力先到顶、显存还富裕
  (15336块是为长上下文准备的,轻负载几乎用不到)。解释了Day17瓶颈是算力非KV
- gpu_prefix_cache_hit_rate=98.96% -> Day18优化的内部证据(外部看TTFT降86%,内部看命中率98.96%)
  命中率是累计量,服务空闲也不归零

## 指标->判断->动作 映射(SRE核心能力)
- 变慢 + waiting飙高 -> 过载,扩容(Golang网关多实例)
- 变慢 + waiting=0 + cache_usage~100% -> KV爆,降max-model-len或减并发
- num_preemptions_total在涨 -> 请求被抢占重算,显存严重不足

## 生产监控链路
vLLM /metrics -> Prometheus定时scrape存时序库 -> Grafana画图+告警
今天curl+grep手动看,理解了整条链路的源头
