package main

import (
"context"
"fmt"
"io"
"time"

"google.golang.org/grpc"
"google.golang.org/grpc/credentials/insecure"

pb "vllm-gateway/proto"
)

// runStream 流式调用:边收边打印,实现打字机效果
func runStream(addr, question string) {
conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
if err != nil {
fmt.Printf("连接失败: %v\n", err)
return
}
defer conn.Close()

client := pb.NewLLMGatewayClient(conn)
ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
defer cancel()

req := &pb.ChatRequest{
Messages: []*pb.Message{
{Role: "system", Content: "你是一个简洁的技术助手。"},
{Role: "user", Content: question},
},
Temperature: 0.0,
MaxTokens:   200,
}

// 发起流式调用,拿到一个流句柄
stream, err := client.ChatStream(ctx, req)
if err != nil {
fmt.Printf("发起流式失败: %v\n", err)
return
}

fmt.Print("流式回答: ")
start := time.Now()
var firstTokenAt time.Duration
gotFirst := false
var instanceID string

// 循环接收每个 chunk
for {
chunk, err := stream.Recv()
if err == io.EOF {
break // 流正常结束
}
if err != nil {
fmt.Printf("\n接收出错: %v\n", err)
return
}
if !gotFirst && chunk.Delta != "" {
firstTokenAt = time.Since(start) // 记录 TTFT
gotFirst = true
}
instanceID = chunk.InstanceId
fmt.Print(chunk.Delta) // 实时打印增量,这就是打字机效果
if chunk.Finished {
break
}
}

fmt.Printf("\n----\n首token耗时(TTFT): %v | 总耗时: %v | 实例: %s\n",
firstTokenAt, time.Since(start), instanceID)
}
