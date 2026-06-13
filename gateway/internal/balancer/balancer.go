package balancer

import (
"sync"
"sync/atomic"

"vllm-gateway/internal/vllm"
)

// Instance 代表一个 vLLM 实例
type Instance struct {
ID     string       // 实例标识,如 "vllm-0" / "vllm-1"
Client *vllm.Client // 对应的 HTTP 客户端
}

// Balancer 负载均衡器:管理实例池 + 选择策略
type Balancer struct {
instances []*Instance
rrCounter uint64     // round-robin 计数器(原子操作,并发安全)
mu        sync.RWMutex
}

func New(instances []*Instance) *Balancer {
return &Balancer{instances: instances}
}

// Pick 用 round-robin 选一个实例
// 用原子自增 % 实例数,保证高并发下分配均匀且无锁竞争
func (b *Balancer) Pick() *Instance {
b.mu.RLock()
defer b.mu.RUnlock()
n := len(b.instances)
if n == 0 {
return nil
}
// atomic 自增,跨 goroutine 安全
idx := atomic.AddUint64(&b.rrCounter, 1)
return b.instances[idx%uint64(n)]
}

// Size 返回当前实例数(后续健康检查会用到)
func (b *Balancer) Size() int {
b.mu.RLock()
defer b.mu.RUnlock()
return len(b.instances)
}
