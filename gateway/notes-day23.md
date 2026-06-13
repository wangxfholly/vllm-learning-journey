# Day 23 — 网关最小闭环:gRPC → HTTP → vLLM 跑通

## 做了什么
1. internal/vllm/client.go:vLLM HTTP 客户端(OpenAI 兼容),屏蔽细节给上层
2. internal/server/server.go:实现 LLMGatewayServer.Chat(),生成 trace_id,转发,回填 instance_id
3. cmd/server/main.go:启动 gRPC server,监听 50051,注册服务
4. cmd/client/main.go:gRPC 测试客户端
5. 三终端联调成功:client → 网关:50051 → vLLM:8000 → GPU → 原路返回

## 实测结果
- 请求"用一句话解释负载均衡",正确返回
- trace_id 网关生成,instance_id 回填,耗时 426ms(网络<1ms,其余全是推理)

## 踩坑
- import 路径错配:vllm-gateway/proto/llm(多了一层)→ 文件在 proto/ 包名 llm
  正确路径是 vllm-gateway/proto,用别名 pb 引用。规律:is not in std = 路径≠module名+目录
- protoc 走 GitHub 超时,改 conda 安装

## 核心认知
- 架构分层:推理框架管「一台机器怎么算」,网关管「一群机器怎么协同+出事怎么办」
- 网络不是 LLM 服务瓶颈(426ms 里网络<1ms),架构决策用数据验证
- 字节内部 vLLM 支持 RPC(Archon)+HTTP,但 LB/failover/降级/限流仍需独立网关层

## 下一步
Day 24:多实例 + 负载均衡,让 instance_id 每次请求都变
