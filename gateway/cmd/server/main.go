package main

import (
"flag"
"log"
"net"
"strings"

"google.golang.org/grpc"

"vllm-gateway/internal/balancer"
"vllm-gateway/internal/server"
"vllm-gateway/internal/vllm"
pb "vllm-gateway/proto"
)

func main() {
grpcAddr := flag.String("addr", ":50051", "gRPC 网关监听地址")
// 用逗号分隔多个 vLLM 后端地址。方案A可填同一个地址两次模拟多实例
backends := flag.String("backends", "http://localhost:8000,http://localhost:8000", "vLLM 实例地址,逗号分隔")
model := flag.String("model", "qwen7b", "served-model-name")
flag.Parse()

// 1. 解析后端列表,构建实例池
urls := strings.Split(*backends, ",")
instances := make([]*balancer.Instance, 0, len(urls))
for i, u := range urls {
u = strings.TrimSpace(u)
instances = append(instances, &balancer.Instance{
ID:     "vllm-" + itoa(i), // vllm-0, vllm-1...
Client: vllm.NewClient(u, *model),
})
log.Printf("注册实例 vllm-%d -> %s", i, u)
}

// 2. 创建负载均衡器 + 网关服务
lb := balancer.New(instances)
gw := server.NewGatewayServer(lb)

// 3. 启动 gRPC server
lis, err := net.Listen("tcp", *grpcAddr)
if err != nil {
log.Fatalf("监听失败: %v", err)
}
grpcServer := grpc.NewServer()
pb.RegisterLLMGatewayServer(grpcServer, gw)

log.Printf("🚀 gRPC 网关启动, 监听 %s, 共 %d 个实例", *grpcAddr, len(instances))
if err := grpcServer.Serve(lis); err != nil {
log.Fatalf("服务启动失败: %v", err)
}
}

// 小工具:int 转 string(避免引入额外包)
func itoa(i int) string {
if i == 0 {
return "0"
}
var b []byte
for i > 0 {
b = append([]byte{byte('0' + i%10)}, b...)
i /= 10
}
return string(b)
}
