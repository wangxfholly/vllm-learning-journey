# Day 25 — 健康检查 + 服务发现摘除

## 做了什么
1. client.go 加两种探活:
   - HealthShallow:GET /health,看进程/端口
   - HealthDeep:发 max_tokens=1 真实推理,抓 GPU OOM/假死(浅探活漏掉的)
2. balancer.go:Instance 加 atomic.Bool 健康状态,Pick() 只选健康实例(扫一圈跳过死的)
3. health/checker.go:后台 goroutine 定时巡检,浅探活→深探活,只在状态变化时打日志
4. main.go:启动健康检查器(间隔5s 超时3s)
5. 实测:8001 不存在 → 几秒内 vllm-1 被自动摘除,流量只走 vllm-0

## 踩坑
- listen tcp :50051: address already in use → 旧网关进程没关,占着端口
  修:lsof -ti :50051 | xargs kill -9。运维高频错误,以后秒判
- cat >> 是追加(跑两次会重复定义),cat > 是覆盖,要分清

## 核心认知
- 浅探活 vs 深探活:LLM 实例会"假死"(端口通但 GPU 爆/卡死),
  只有深探活(真实推理)能抓住,这是 LLM 网关比普通网关多做的一层
- 健康状态用 atomic.Bool:检查器写、Pick 读,并发访问必须原子,否则 data race
- 日志只报状态变化(prev!=now),避免健康实例每5秒刷屏

## 下一步
Day 26:failover + 降级。摘除只能挡"已知死的",
请求打到一半实例突然挂怎么办?这是 LLM 网关最难的(尤其流式中途失败)
