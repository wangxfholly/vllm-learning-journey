# Day 29 —— 混沌测试:亲手杀掉 vLLM,看系统怎么死、怎么活

## 一、今天干了什么
不再是"我以为它能扛",而是**在压力下亲手制造故障**,验证 Day 25-26 写的容错机制(摘除/failover/降级)在真实并发下到底工不工作、能不能自动恢复。

## 二、混沌测试设计(单卡约束下的取巧)
单 GPU 跑不了两个真 vLLM 实例,于是:
- 真实例:vllm-0 → localhost:8000(真 vLLM)
- 制造故障:压测中途 `pkill` 掉 vLLM 进程,模拟实例突然死亡
- 用 cmd/bench 并发压测器(-c 4)持续打 Chat,让降级/恢复在每秒输出里实时可见

四幕剧:①正常压测 → ②观察健康基线 → ③杀掉 vLLM → ④等待健康检查自动复活

## 三、Chaos Engineering 核心思想
Netflix Chaos Monkey:主动制造故障看系统怎么死。
> "90% 的容灾机制,第一次真出事时根本不工作。"
混沌测试的价值 = 在可控环境里,把"我以为它能扛"变成"我看见它扛住了"。

## 四、实测结果(体检报告)
| 指标 | 值 | 含义 |
|------|-----|------|
| gateway_requests_total{vllm-0, ok} | 844 | 健康期真实推理成功 |
| gateway_failover_total | 4 | 只有 4 个请求亲历"健康→死亡"瞬间,触发了 failover 重试 |
| gateway_requests_total{all_failed, degraded} | 4 | 这 4 个重试全失败后降级 |
| gateway_requests_total{none, degraded} | 764546 | vllm-0 被标记 unhealthy 后,直接走空实例降级路径(不再进 for 循环) |
| gateway_degraded_total | 764550 | = 4 + 764546,降级总数 |
| ❌ 失败 | 0 | 全程零 RPC error |
| gateway_healthy_instances | 1 | 测试结束已自动恢复 |

## 五、关键洞察
### 1. failover_total=4 vs degraded=76万 —— 被动摘除在保护系统
- failover_total=4 不是"只重试了 4 次",而是只有 4 个请求"亲历"了从健康到死亡的瞬间。
- 一旦 SetHealthy(false) 生效,后续请求连重试都省了,被摘除路径直接挡住 → 避免 76 万次无谓的网络超时,防止雪崩。
- instance="all_failed"(4) vs instance="none"(764546):精确区分"实例存在但全挂"和"压根没有可用实例"两种降级。这是 Day 28 给 RequestsTotal 打 instance 标签的回报。

### 2. QPS 暴涨是危险信号,不是好事
- 降级期 QPS 飙到 23000+:因为降级响应本地拼字符串就返回(~0ms),不走推理。
- 恢复期 QPS 暴跌到 8-12:这才是单卡真实推理水位。
- 教训:只看 QPS 不看 status 标签,会被"后端全挂在疯狂返错"伪装成"吞吐暴涨"骗到。

### 3. 容错三级阶梯在压力下接力成功
在途失败 → ①failover 重试 → 全失败 → ②被动摘除不再撞死实例 → 无可用 → ③降级兜底(失败=0)→ vLLM 复活 → 健康检查探活自动纳入(成功重新增长,降级冻结)。
全程自动恢复,无需手动重启网关。

## 六、踩坑
- CUDA OOM:重启 vLLM 报 OutOfMemory,实为僵尸进程占着 19G 显存。
  教训:OOM 先 `nvidia-smi` 查 PID → `pkill` 清僵尸 → 再启动,比盲调 gpu-memory-utilization 对症。
- 代理:外网下载用 proxy-on(sys-proxy-rd-relay.byted.org:8118),用完立刻 proxy-off,
  否则 no_proxy 不覆盖 localhost,会打断本地 gateway↔vLLM↔gRPC 环回调用。
