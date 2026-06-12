# Day 06 - Week 1 复盘

## 本周完成
- Day 01 环境搭建 + hello_vllm(__file__ 路径,67.61 tok/s)
- Day 02 批量 vs 串行(8.48x)+ 在线服务 + OpenAI SDK
- Day 03 采样参数(temperature/top_p/seed/repetition_penalty)
- Day 04 并发压测(11.6x)+ 流式(TTFT 0.292s, TPOT 14.1ms)
- Day 05 量化(AWQ 省 62% 显存,2.1x 吞吐)
- Day 06 技术博客 + 复盘

## 核心闭环
原理(prefill/decode)→ 指标(TTFT/TPOT/吞吐)→ 实测数据

## Week 2 计划
- 量化跑 7B/14B
- speculative decoding
- prefill/decode 分离部署
