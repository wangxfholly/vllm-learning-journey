# Day 04 — 并发压测 & 流式输出

## 三个生产核心指标
- TTFT (Time To First Token):发请求到首个字的时间,对应 prefill 阶段,衡量等待感
- TPOT (Time Per Output Token):每个后续 token 间隔,对应 decode 阶段,衡量吐字流畅度
- Throughput:系统每秒总 token,衡量成本/容量

## 实验① 并发压测(容量曲线)
| 并发 | 系统吞吐 | 平均延迟 |
|---|---|---|
| 1  | 69.6 tok/s | 0.88s |
| 5  | 266.0 | 1.24s |
| 10 | 451.5 | 1.31s |
| 20 | 811.1 | 1.42s |

结论:并发 1→20,吞吐涨 11.6x,延迟仅涨 1.6x。几乎同样等待时间服务 20 倍用户=continuous batching 的价值。吞吐边际收益递减,继续加并发会触天花板,延迟急剧恶化(排队)。生产定容量=找"延迟没爆、吞吐够高"的甜点。

## 实验② 流式输出(stream=True)
- TTFT=0.292s,总耗时1.515s,TPOT=14.1ms/chunk
- 流式意义:非流式干等1.5s,流式0.29s见首字。同样总时间,等待焦虑大降
- stream=True → SSE 边生成边推;返回 delta(增量)不是完整 message
- TTFT 衡量 prefill,TPOT 衡量 decode → 与 Day1 理论闭环

## 生产 SLA 经验值
- 对话类:TTFT < 500ms,TPOT < 50ms
- L4 跑 1.5B:TTFT 292ms / TPOT 14ms,达标
- 模型变大指标会涨 → 靠量化/更好的卡/投机解码压回

## 认知闭环
请求→Prefill(定TTFT)→Decode(定TPOT);多请求→Continuous Batching(定吞吐,并发↑吞吐↑延迟↑)

## 明日(Day 05)
- 量化初探:为什么要量化,FP16 vs AWQ/GPTQ/FP8,显存与速度对比
- 或:Week 1 复盘 + 把一周成果整理成博客草稿
