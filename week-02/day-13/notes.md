# Day 13 - Week 2 复盘

## 本周完成(Day 07-13)
- Day 07 单卡跑 7B AWQ,显存账(权重24% / KV Cache 60%)
- Day 08 KV Cache 深挖,长上下文双重打击(激活+KV)
- Day 09 投机解码,vocab_size 坑 + n-gram 无损加速
- Day 10 并发调优 max_num_seqs(7.8x),边际递减
- Day 11 guided decoding,工具调用 100% 合法 JSON
- Day 12 综合项目 Mini Agent(端到端工具调用)
- Day 13 复盘 + 博客

## 一句话核心
量化换产能、KV Cache 是显存大头、投机解码无损但挑场景、max_num_seqs 是
吞吐/延迟旋钮、guided decoding 锁死输出是 Agent 生产化关键。

## 最大认知升级
推理引擎不是黑盒。每个性能数字背后都有可解释原因，这是上层框架封装掉、
但出问题必须懂的底层。

## Week 3 计划(候选)
- 多卡张量并行(tensor parallel)跑 14B/32B
- prefill/decode 分离部署
- Mini Agent 接入真实业务/接 OpenAI 兼容服务
