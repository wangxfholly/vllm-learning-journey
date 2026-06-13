package server

import (
"context"
"fmt"
"log"

"github.com/google/uuid"

"vllm-gateway/internal/balancer"
"vllm-gateway/internal/vllm"
pb "vllm-gateway/proto"
)

// GatewayServer 实现 pb.LLMGatewayServer 接口
type GatewayServer struct {
pb.UnimplementedLLMGatewayServer
lb *balancer.Balancer // 不再是单个 client,而是负载均衡器
}

func NewGatewayServer(lb *balancer.Balancer) *GatewayServer {
return &GatewayServer{lb: lb}
}

// Chat 实现非流式接口
func (s *GatewayServer) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
traceID := req.TraceId
if traceID == "" {
traceID = uuid.NewString()
}

// 1. 负载均衡:选一个实例
inst := s.lb.Pick()
if inst == nil {
return nil, fmt.Errorf("无可用 vLLM 实例")
}
log.Printf("[Chat] trace_id=%s 路由到实例=%s, messages=%d", traceID, inst.ID, len(req.Messages))

// 2. 转换消息格式
pairs := make([][2]string, 0, len(req.Messages))
for _, m := range req.Messages {
pairs = append(pairs, [2]string{m.Role, m.Content})
}
messages := vllm.BuildMessages(pairs)

// 3. 调被选中的实例
result, err := inst.Client.Chat(ctx, messages, req.Temperature, req.MaxTokens)
if err != nil {
log.Printf("[Chat] trace_id=%s 实例=%s 调用失败: %v", traceID, inst.ID, err)
return nil, err
}

// 4. 回填 instance_id = 被选中的实例 ID
log.Printf("[Chat] trace_id=%s 实例=%s 成功, completion_tokens=%d", traceID, inst.ID, result.CompletionTokens)
return &pb.ChatResponse{
Content:          result.Content,
TraceId:          traceID,
InstanceId:       inst.ID,
PromptTokens:     result.PromptTokens,
CompletionTokens: result.CompletionTokens,
}, nil
}
