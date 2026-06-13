package main

import (
"context"
"flag"
"log"
"time"

"google.golang.org/grpc"
"google.golang.org/grpc/credentials/insecure"

pb "vllm-gateway/proto"
)

func main() {
addr := flag.String("addr", "localhost:50051", "网关地址")
question := flag.String("q", "用一句话解释什么是负载均衡", "问题")
streamMode := flag.Bool("stream", false, "是否用流式")
flag.Parse()

if *streamMode {
runStream(*addr, *question) // 流式模式(在 stream.go)
return
}

// 非流式模式
conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
if err != nil {
log.Fatalf("连接网关失败: %v", err)
}
defer conn.Close()

client := pb.NewLLMGatewayClient(conn)
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

req := &pb.ChatRequest{
Messages: []*pb.Message{
{Role: "system", Content: "你是一个简洁的技术助手。"},
{Role: "user", Content: *question},
},
Temperature: 0.0,
MaxTokens:   128,
}

start := time.Now()
resp, err := client.Chat(ctx, req)
if err != nil {
log.Fatalf("调用 Chat 失败: %v", err)
}
log.Printf("===== 网关返回 =====")
log.Printf("内容: %s", resp.Content)
log.Printf("trace_id: %s", resp.TraceId)
log.Printf("instance_id: %s", resp.InstanceId)
log.Printf("prompt_tokens=%d completion_tokens=%d", resp.PromptTokens, resp.CompletionTokens)
log.Printf("耗时: %v", time.Since(start))
}
