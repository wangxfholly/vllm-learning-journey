# Day 22 — 阶段二开篇:gRPC 网关 proto 契约设计

## 做了什么
1. 设计 llm.proto:定义 LLMGateway 服务,含 Chat(unary)+ ChatStream(server-streaming)
2. 关键设计:流式用 delta 增量(非全文重传);提前埋 trace_id / instance_id 可观测字段
3. 装工具链:protoc(conda)+ protoc-gen-go + protoc-gen-go-grpc
4. 编译生成 llm.pb.go(message→struct)+ llm_grpc.pb.go(service→接口/stub)

## 踩坑
- protoc 走 GitHub releases 下载超时(GitHub 不通)→ 改用 conda-forge 安装,绕开
- Go 依赖被 grpc v1.81.1 要求自动从 1.21 升到 1.25,正常,toolchain 自动管理

## 核心认知
- proto = 契约;gRPC 生成「客户端完整实现 + 服务端接口空壳」
- 网关全部业务逻辑(选实例/转发/failover)写在服务端实现里
- 接口设计要为未来留口子(trace 字段现在埋,别返工)

## 架构
对外:标准 gRPC + 流式  |  对内:HTTP 调 vLLM OpenAI 接口
网关 = gRPC-to-HTTP 智能代理 + 高可用控制面
