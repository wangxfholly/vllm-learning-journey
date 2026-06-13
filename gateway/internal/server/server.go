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

// ChatStream 实现流式接口:vLLM SSE channel -> gRPC stream,边收边发
// 策略B:首 token 前失败可 failover;首 token 后失败只能优雅断流(避免内容重复)
func (s *GatewayServer) ChatStream(req *pb.ChatRequest, stream pb.LLMGateway_ChatStreamServer) error {
ctx := stream.Context()
traceID := req.TraceId
if traceID == "" {
traceID = uuid.NewString()
}

pairs := make([][2]string, 0, len(req.Messages))
for _, m := range req.Messages {
pairs = append(pairs, [2]string{m.Role, m.Content})
}
messages := vllm.BuildMessages(pairs)

order := s.lb.PickHealthyOrder()
if len(order) == 0 {
log.Printf("[Stream] trace_id=%s 无健康实例,降级", traceID)
return stream.Send(&pb.ChatStreamChunk{
Delta: degradeMessage, Finished: true, TraceId: traceID, InstanceId: "degraded",
})
}

// 遍历健康实例,但 failover 只在"还没吐过任何 token"时允许
var lastErr error
for attempt, inst := range order {
sentAnyToken := false // 本次尝试是否已经向客户端发过 token

ch, err := inst.Client.ChatStream(ctx, messages, req.Temperature, req.MaxTokens)
if err != nil {
// 连接阶段就失败,还没吐 token,可以 failover
lastErr = err
inst.SetHealthy(false)
log.Printf("[Stream] trace_id=%s 实例=%s 连接失败,failover: %v", traceID, inst.ID, err)
continue
}
log.Printf("[Stream] trace_id=%s 第%d次尝试,实例=%s 开始流式", traceID, attempt+1, inst.ID)

streamErr := false
for chunk := range ch {
if chunk.Err != nil {
// 流中途出错
lastErr = chunk.Err
streamErr = true
log.Printf("[Stream] trace_id=%s 实例=%s 流中途失败: %v", traceID, inst.ID, chunk.Err)
break
}
if chunk.Finished {
// 正常结束
log.Printf("[Stream] trace_id=%s 实例=%s 流正常结束", traceID, inst.ID)
return stream.Send(&pb.ChatStreamChunk{
Finished: true, TraceId: traceID, InstanceId: inst.ID,
})
}
// 转发一个 delta 给客户端
if err := stream.Send(&pb.ChatStreamChunk{
Delta: chunk.Delta, TraceId: traceID, InstanceId: inst.ID,
}); err != nil {
return err // 客户端断开,直接结束
}
sentAnyToken = true
}

if streamErr {
inst.SetHealthy(false)
if sentAnyToken {
// 已经吐过 token,不能 failover(会重复),只能优雅断流
log.Printf("[Stream] trace_id=%s 已吐token后失败,优雅断流(不failover)", traceID)
return stream.Send(&pb.ChatStreamChunk{
Delta: "\n[服务中断,请重试]", Finished: true, TraceId: traceID, InstanceId: inst.ID,
})
}
// 还没吐 token,可以 failover 到下一个实例
log.Printf("[Stream] trace_id=%s 首token前失败,failover", traceID)
continue
}
}

// 所有实例都失败且都没吐过 token -> 降级
log.Printf("[Stream] trace_id=%s 全部失败,降级(最后错误: %v)", traceID, lastErr)
return stream.Send(&pb.ChatStreamChunk{
Delta: degradeMessage, Finished: true, TraceId: traceID, InstanceId: "degraded",
})
}
