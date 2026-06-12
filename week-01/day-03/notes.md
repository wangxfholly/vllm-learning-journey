# Day 03 — 采样参数:控制模型怎么说话

## 核心原理
模型每步在几万候选词上算概率分布,采样参数控制"怎么挑":
- temperature:缩放分布陡峭度。低=尖(选最可能),高=平(给小概率词机会)
- top_p(核采样):只在累积概率前 p 的候选里挑,砍长尾
- top_k:只在概率最高 k 个里挑
- repetition_penalty:对已出现 token 降权,防复读

## 实验现象
### ① temperature
- 0.0:三次"咖啡星尘"完全一样 → 贪心解码,可复现
- 0.7:2 同 1 异,适度多样
- 1.5:"星萃安toBeInTheDocument" 等,中英混杂语法崩坏 → 高温把离谱 token 也采出来

### ② temperature=1.5 + top_p=0.5
- 输出恢复正常("蒸汽晨曦咖啡馆")→ top_p 砍掉长尾离谱词
- 生产黄金搭配:temperature 0.7-0.9 + top_p 0.9

### ③ temperature=0 + seed=42
- 确定性输出。seed 真正价值:temperature>0 时也能复现随机序列,便于 debug/AB

### ④ repetition_penalty
- 短输出(我爱编程。)看不出差异,正常
- 主要解决长文本复读/循环,penalty 1.1-1.3

## 参数配方速查
| 需求 | 配方 |
|---|---|
| 确定可复现(分类/抽取/function call) | temperature=0 |
| 创意不离谱(文案/对话) | temperature 0.7-0.9 + top_p 0.9 |
| 可复现的随机(测试) | temperature>0 + 固定 seed |
| 治长文复读 | repetition_penalty 1.1-1.3 |

## 明日(Day 04)
- server 并发压测:同时打多个请求,观察 continuous batching 自动拼批
- 或:流式输出 stream=True(打字机效果)+ 首 token 延迟(TTFT)概念
