# Day 28:可观测性双柱 —— Trace 链路追踪 + Metrics 指标

## 今日目标
把网关一路埋的 trace_id,升级成生产级可观测性:OpenTelemetry Trace + Prometheus Metrics。

## 核心认知:Trace vs Metrics 分工
- Metrics(指标):回答"系统整体健康吗"——聚合数字,告警基于它,先报警
- Trace(链路):回答"这一次请求慢在哪"——单请求调用树,定位根因,后钻取
- 标准动线:Metrics 发现异常 → 圈定时间窗 → Trace 钻进去看根因

## 可观测性三支柱
1. Metrics —— Prometheus 格式,本次实现
2. Tracing —— OpenTelemetry + (Jaeger),本次实现埋点
3. Logging —— 之前的 trace_id 日志

## Part 1:Trace(OpenTelemetry)
- Trace = 一整条链路(唯一 TraceID);Span = 链路中一段(有起止时间,可嵌套)
- Context Propagation = trace 上下文跨进程传递(W3C traceparent header)
- 实现:internal/telemetry/tracer.go
  - InitTracer:环境变量 OTEL_OTLP_ENDPOINT 切换 exporter
    - 空 -> stdout(开发期,控制台打印 trace 树)
    - 有值 -> OTLP gRPC(接 Jaeger localhost:4317)
  - 体现 OTel 设计哲学:埋点与后端解耦,切换只改一行
- server.go Chat 埋三个 span:
  - gateway.Chat(根)
  - lb.PickHealthyOrder(子)
  - vllm.Chat(子,带 instance.id 属性)
- 父子关系靠 SpanID / ParentSpanID 串联,同一 TraceID = 同一棵树

## 时间归因实测(本次请求)
- gateway.Chat 总:2557.1 ms
- lb.PickHealthyOrder:0.0015 ms(1.5 微秒)
- vllm.Chat:2557.0 ms(99.99%)
- 结论:慢在 vLLM 推理,不在网关。纯 grep 日志做不到这种拆分。

## Part 2:Metrics(Prometheus client_golang)
- 选型:Trace 用 OTel,Metrics 用 Prometheus client_golang(后端最主流组合)
- internal/telemetry/metrics.go,/metrics 端点监听 :2112
- 通用层(RED):
  - gateway_requests_total (Counter, labels: method/status/instance)
  - gateway_request_duration_seconds (Histogram)
- LLM 专属层:
  - gateway_ttft_seconds (Histogram) —— 首 token 耗时
  - gateway_healthy_instances (Gauge) —— 实时健康实例数,告警源
  - gateway_failover_total (Counter)
  - gateway_degraded_total (Counter)
  - gateway_tokens_total (Counter)

## Prometheus 关键概念(踩坑点)
- Counter 只增不减,要的是它的"增长速率":rate(xxx_total[1m]) = QPS
- Histogram 的 _bucket 用 histogram_quantile(0.99,...) 算 P99
- Gauge 可增可减,直接画曲线,< 阈值即告警
- label 是精华:sum by (instance/status) 可任意下钻定位问题

## 实测验证
- 发 3 个请求后:
  - gateway_requests_total{instance="vllm-0",method="chat",status="ok"} 3
  - gateway_healthy_instances 1
  - gateway_tokens_total 384 (= 3 × 128)
  - gateway_request_duration_seconds_count{method="chat"} 3

## 遗留(Step 4,待 Jaeger/Prometheus 装好)
- traceparent header 传 vLLM,串起 vLLM 内部 prefill/decode span
- Jaeger UI 看火焰图(后台下载中,网络受限)
- Prometheus 抓取 + Grafana 画 dashboard

## 环境踩坑
- GitHub / ghproxy 在开发机不通,只有 GOPROXY/conda/pip 镜像通
- 公司代理 sys-proxy-rd-relay.byted.org:8118 可翻外网,但用完即关
  (no_proxy 只配了 .byted.org,不关会影响 localhost 本地回环)
- go get 必须在有 go.mod 的目录(gateway/)里跑
- main.go 漏调 InitTracer -> Tracer 为 nil -> 请求时 nil panic 崩端口
