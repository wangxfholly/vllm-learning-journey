package health

import (
"context"
"log"
"time"

"vllm-gateway/internal/balancer"
)

// Checker 后台健康检查器
type Checker struct {
lb       *balancer.Balancer
interval time.Duration
timeout  time.Duration
}

func New(lb *balancer.Balancer, interval, timeout time.Duration) *Checker {
return &Checker{lb: lb, interval: interval, timeout: timeout}
}

// Start 启动后台巡检(在独立 goroutine 里跑)
func (c *Checker) Start(ctx context.Context) {
go func() {
ticker := time.NewTicker(c.interval)
defer ticker.Stop()
// 启动先立刻探一次,不等第一个 interval
c.checkAll(ctx)
for {
select {
case <-ctx.Done():
return
case <-ticker.C:
c.checkAll(ctx)
}
}
}()
}

// checkAll 巡检所有实例:浅探活 -> 深探活
func (c *Checker) checkAll(ctx context.Context) {
for _, inst := range c.lb.All() {
probeCtx, cancel := context.WithTimeout(ctx, c.timeout)
err := inst.Client.HealthShallow(probeCtx)
if err == nil {
// 浅探活过了,再做深探活
err = inst.Client.HealthDeep(probeCtx)
}
cancel()

prev := inst.IsHealthy()
now := err == nil
inst.SetHealthy(now)

// 只在状态发生变化时打日志,避免刷屏
if prev != now {
if now {
log.Printf("✅ [健康检查] 实例 %s 恢复健康,重新加入实例池", inst.ID)
} else {
log.Printf("❌ [健康检查] 实例 %s 不健康,已摘除! 原因: %v", inst.ID, err)
}
}
}
}
