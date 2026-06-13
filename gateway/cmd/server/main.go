package main

import (
"flag"
"log"
"net"

"google.golang.org/grpc"

"vllm-gateway/internal/server"
"vllm-gateway/internal/vllm"
pb "vllm-gateway/proto"
)

func main() {
// 命令行参数:网关监听端口、vLLM 地址、模型名
grpcAddr := flag.String("addr", ":50051", "gRPC 网关监听地址")
vllmURL := flag.String("vllm", "http://localhost:8000", "vLLM 实例地址")
model := flag.String("model", "qwen7b", "served-model-name")
flag.Parse()

// 1. 创建 vLLM 客户端
client := vllm.NewClient(*vllmURL, *model)

// 2. 创建网关服务实现
gw := server.NewGatewayServer(client, *vllmURL)

// 3. 启动 gRPC server
lis, err := net.Listen("tcp", *grpcAddr)
if err != nil {
log.Fatalf("监听失败: %v", err)
}
grpcServer := grpc.NewServer()
pb.RegisterLLMGatewayServer(grpcServer, gw) // 把实现注册进去

log.Printf("🚀 gRPC 网关启动, 监听 %s, 后端 vLLM=%s, model=%s", *grpcAddr, *vllmURL, *model)
if err := grpcServer.Serve(lis); err != nil {
log.Fatalf("服务启动失败: %v", err)
}
}
