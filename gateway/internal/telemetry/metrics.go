package telemetry

import (
"github.com/prometheus/client_golang/prometheus"
"github.com/prometheus/client_golang/prometheus/promauto"
)

// 通用层(RED)
var (
// 总请求数,按 method(chat/stream)、status(ok/degraded)、instance 打标签
RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
Name: "gateway_requests_total",
Help: "网关处理的总请求数",
}, []string{"method", "status", "instance"})

// 请求耗时分布(算 P50/P99 的基础)
RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
Name:    "gateway_request_duration_seconds",
Help:    "网关请求耗时(秒)",
Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30},
}, []string{"method"})
)

// LLM 专属层
var (
// 首 token 耗时(TTFT)—— 流式才有
TTFT = promauto.NewHistogram(prometheus.HistogramOpts{
Name:    "gateway_ttft_seconds",
Help:    "首 token 耗时 TTFT(秒)",
Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
})

// 当前健康实例数(Gauge,实时反映健康检查/摘除)
HealthyInstances = promauto.NewGauge(prometheus.GaugeOpts{
Name: "gateway_healthy_instances",
Help: "当前健康的 vLLM 实例数",
})

// failover 触发次数
FailoverTotal = promauto.NewCounter(prometheus.CounterOpts{
Name: "gateway_failover_total",
Help: "failover 触发的总次数",
})

// 降级触发次数
DegradedTotal = promauto.NewCounter(prometheus.CounterOpts{
Name: "gateway_degraded_total",
Help: "降级触发的总次数",
})

// 输出 token 总量
TokensTotal = promauto.NewCounter(prometheus.CounterOpts{
Name: "gateway_tokens_total",
Help: "输出 completion token 总量",
})
)
