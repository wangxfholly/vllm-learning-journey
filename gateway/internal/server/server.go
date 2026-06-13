package server

import (
"context"
"log"

"github.com/google/uuid"

"vllm-gateway/internal/vllm"
pb "vllm-gateway/proto"
)

// GatewayServer 实现 pb.LLMGatewayServer 接口
type GatewayServer struct {
pb.UnimplementedLLMGatewayServer // 嵌入空实现,未实现的方法自动有默认行为
vllmClient                       *vllm.Client
instanceID                       string // 当前转发到的 vLLM 实例标识(Day23 只有一个)
}

func NewGatewayServer(client *vllm.Client, instanceID string) *GatewayServer {
return &GatewayServer{
vllmClient: client,
instanceID: instanceID,
}
}

// Chat 实现非流式接口:gRPC 入参 -> 调 vLLM -> 填 gRPC 返回
func (s *GatewayServer) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
// 1. trace_id:客户端没传就生成一个(为后续链路追踪埋点)
traceID := req.TraceId
if traceID == "" {
traceID = uuid.NewString()
}
log.Printf("[Chat] trace_id=%s instance=%s 收到请求, messages=%d", traceID, s.instanceID, len(req.Messages))

// 2. 把 gRPC 消息转成 vLLM client 需要的格式
pairs := make([][2]string, 0, len(req.Messages))
for _, m := range req.Messages {
pairs = append(pairs, [2]string{m.Role, m.Content})
}
messages := vllm.BuildMessages(pairs)

// 3. 调 vLLM
result, err := s.vllmClient.Chat(ctx, messages, req.Temperature, req.MaxTokens)
if err != nil {
log.Printf("[Chat] trace_id=%s 调用 vLLM 失败: %v", traceID, err)
return nil, err
}

// 4. 填 gRPC 返回(把 instance_id / trace_id 回传,便于排障)
log.Printf("[Chat] trace_id=%s 成功, completion_tokens=%d", traceID, result.CompletionTokens)
return &pb.ChatResponse{
Content:          result.Content,
TraceId:          traceID,
InstanceId:       s.instanceID,
PromptTokens:     result.PromptTokens,
CompletionTokens: result.CompletionTokens,
}, nil
}
