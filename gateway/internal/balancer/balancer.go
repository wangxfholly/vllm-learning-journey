package balancer

import (
"sync"
"sync/atomic"

"vllm-gateway/internal/vllm"
)

// Instance 代表一个 vLLM 实例
type Instance struct {
ID      string
Client  *vllm.Client
healthy atomic.Bool // 健康状态(原子,跨 goroutine 安全读写)
}

// SetHealthy 由健康检查器调用
func (i *Instance) SetHealthy(ok bool) { i.healthy.Store(ok) }

// IsHealthy 供 Pick / 监控查询
func (i *Instance) IsHealthy() bool { return i.healthy.Load() }

// Balancer 负载均衡器
type Balancer struct {
instances []*Instance
rrCounter uint64
mu        sync.RWMutex
}

func New(instances []*Instance) *Balancer {
// 初始假定健康,等首轮探活修正
for _, inst := range instances {
inst.SetHealthy(true)
}
return &Balancer{instances: instances}
}

// Pick 只从健康实例里 round-robin 选择
// 做法:轮询起点用原子自增,然后最多扫一圈找到第一个健康实例
func (b *Balancer) Pick() *Instance {
b.mu.RLock()
defer b.mu.RUnlock()
n := len(b.instances)
if n == 0 {
return nil
}
start := atomic.AddUint64(&b.rrCounter, 1)
// 从 start 开始最多扫一圈,跳过不健康的
for offset := 0; offset < n; offset++ {
idx := (start + uint64(offset)) % uint64(n)
inst := b.instances[idx]
if inst.IsHealthy() {
return inst
}
}
return nil // 全挂了
}

// All 返回所有实例(健康检查器遍历用)
func (b *Balancer) All() []*Instance {
b.mu.RLock()
defer b.mu.RUnlock()
out := make([]*Instance, len(b.instances))
copy(out, b.instances)
return out
}

// HealthySize 返回当前健康实例数(监控用)
func (b *Balancer) HealthySize() int {
b.mu.RLock()
defer b.mu.RUnlock()
cnt := 0
for _, inst := range b.instances {
if inst.IsHealthy() {
cnt++
}
}
return cnt
}

// PickHealthyOrder 返回一个健康实例的有序列表,用于 failover:
// 先按 round-robin 选出起点,再把其余健康实例依次排在后面。
// server 拿到这个列表后,从头到尾依次尝试,直到某个成功。
func (b *Balancer) PickHealthyOrder() []*Instance {
b.mu.RLock()
defer b.mu.RUnlock()
n := len(b.instances)
if n == 0 {
return nil
}
start := atomic.AddUint64(&b.rrCounter, 1)
order := make([]*Instance, 0, n)
for offset := 0; offset < n; offset++ {
idx := (start + uint64(offset)) % uint64(n)
inst := b.instances[idx]
if inst.IsHealthy() {
order = append(order, inst)
}
}
return order
}
