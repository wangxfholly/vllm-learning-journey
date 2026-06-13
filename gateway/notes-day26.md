# Day 26 — Failover 故障转移 + 降级兜底

## 概念厘清(关键)
- Failover 故障转移:某实例失败 → 平级换另一个健康实例重试,质量不变,用户无感
- 降级 Degradation:所有实例都不行 → 牺牲质量保可用,返回兜底话术,用户有感但不报错
- 容错三级阶梯:LB 正常分发 → 失败 failover 换实例 → 全挂降级兜底

## 做了什么
1. balancer.go 加 PickHealthyOrder():返回健康实例有序列表(rr起点+其余),供 failover 依次尝试
2. server.go Chat 改造:
   - 遍历健康实例列表,逐个尝试,成功即返回
   - 失败立即 SetHealthy(false) 被动摘除(不等下轮健康检查)
   - 全部失败 → degradeResponse 返回兜底话术,instance_id=degraded
3. 降级返回 nil error + 兜底内容(非 gRPC error),用户体验更好

## 实测
- 降级:两后端都不存在 → 返回"抱歉,服务当前繁忙",instance_id=degraded,耗时3ms(不碰GPU)
- failover:8001死 → 轮到也自动转到 vllm-0,返回正常答案

## LLM 特殊性
- 非流式失败可安全重试(还没返回内容)→ 今天做的
- 流式中途失败不能简单重试(会重复内容)→ Day27 专门处理

## 核心认知
- 主动健康检查(Day25 探活)+ 被动摘除(Day26 请求失败即摘)互补:
  主动发现"还没人撞上的死",被动处理"刚撞上的死"
- 降级路径必须又快又轻,系统崩溃时不能再拖慢后端

## 下一步
Day 27:server-streaming 流式打字机 + 流式中途失败处理
