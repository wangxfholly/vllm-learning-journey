# Day 27 — 流式(server-streaming)+ 流式中途失败处理【阶段二最难】

## 做了什么
1. client.go 加 ChatStream:vLLM SSE 流(data: {...} / [DONE]),bufio 边读边解析,
   通过 channel 吐 StreamChunk{Delta, Finished, Err}
2. server.go 加 ChatStream:实现 gRPC server-streaming,vLLM channel -> stream.Send 管道转发
3. cmd/client/stream.go + main.go -stream 开关:Recv 循环实时打印,打字机效果
4. 实测:三句话逐字流出,TTFT 27ms,总耗时 1.08s,实例 vllm-0

## 流式中途失败(本日核心难点)
- 矛盾:token 发出去收不回,失败后 failover 会导致内容重复
- 三种策略:A直接断流报错 / B首token前可failover后只能断流(最常用) / C客户端重试
- 实现策略B:sentAnyToken 布尔做"首token分水岭"
  - 首token前失败 → continue 安全 failover
  - 首token后失败 → 优雅断流"[服务中断,请重试]",不 failover

## 踩坑
- Go import 必须在文件顶部,中间不能写;cat>> 追加代码后缺 import
- 解决:goimports 自动补全(go run golang.org/x/tools/cmd/goimports@latest -w),
  比手动 sed 可靠,养成"保存即 goimports"习惯

## 两阶段闭环
- 阶段一 prefix caching 把 TTFT 122→29ms;今天网关流式实测 TTFT 27ms
- 网关零损耗(双重转发不增加可感知延迟),再次实证"网络不是瓶颈"
- 流式让感知延迟=27ms 而非1080ms,体感快40倍

## 核心认知
- 管道转发:边收 vLLM 边 Send 客户端,全程不缓存全文
- stream.Context() 监听客户端断开,中途断开就停,不白耗 GPU
- 首token是流式容错的分水岭:之前等同非流式,之后回不了头

## 下一步
Day 28:trace 链路监控接 Jaeger(OpenTelemetry),让 trace_id 变成可视化调用链
