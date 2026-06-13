package server

import (
"context"
"log"

"github.com/google/uuid"

"vllm-gateway/internal/balancer"
"vllm-gateway/internal/vllm"
pb "vllm-gateway/proto"
)

// 降级兜底话术:所有实例都失败时返回,保证用户不是看到冷冰冰的 error
const degradeMessage = "抱歉,服务当前繁忙,请稍后重试。"

type GatewayServer struct {
pb.UnimplementedLLMGatewayServer
lb *balancer.Balancer
}

func NewGatewayServer(lb *balancer.Balancer) *GatewayServer {
return &GatewayServer{lb: lb}
}

func (s *GatewayServer) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
traceID := req.TraceId
if traceID == "" {
traceID = uuid.NewString()
}

// 转换消息
pairs := make([][2]string, 0, len(req.Messages))
for _, m := range req.Messages {
pairs = append(pairs, [2]string{m.Role, m.Content})
}
messages := vllm.BuildMessages(pairs)

// 1. 拿到健康实例的有序列表(failover 用)
order := s.lb.PickHealthyOrder()
if len(order) == 0 {
// 没有任何健康实例 -> 直接降级
log.Printf("[Chat] trace_id=%s 无健康实例,触发降级", traceID)
return s.degradeResponse(traceID), nil
}

// 2. 依次尝试每个健康实例(failover)
var lastErr error
for attempt, inst := range order {
log.Printf("[Chat] trace_id=%s 第%d次尝试,实例=%s", traceID, attempt+1, inst.ID)
result, err := inst.Client.Chat(ctx, messages, req.Temperature, req.MaxTokens)
if err == nil {
// 成功!
log.Printf("[Chat] trace_id=%s 实例=%s 成功(第%d次尝试)", traceID, inst.ID, attempt+1)
return &pb.ChatResponse{
Content:          result.Content,
TraceId:          traceID,
InstanceId:       inst.ID,
PromptTokens:     result.PromptTokens,
CompletionTokens: result.CompletionTokens,
}, nil
}
// 失败,记下来,标记该实例不健康,继续 failover 到下一个
lastErr = err
inst.SetHealthy(false) // 调用失败立即摘除,不等下一轮健康检查
log.Printf("[Chat] trace_id=%s 实例=%s 失败: %v,failover 到下一个", traceID, inst.ID, err)
}

// 3. 所有健康实例都试过了还是失败 -> 降级
log.Printf("[Chat] trace_id=%s 所有实例 failover 失败(最后错误: %v),触发降级", traceID, lastErr)
return s.degradeResponse(traceID), nil
}

// degradeResponse 构造降级兜底响应
func (s *GatewayServer) degradeResponse(traceID string) *pb.ChatResponse {
return &pb.ChatResponse{
Content:    degradeMessage,
TraceId:    traceID,
InstanceId: "degraded", // 标记这是降级响应,便于监控统计降级率
}
}
