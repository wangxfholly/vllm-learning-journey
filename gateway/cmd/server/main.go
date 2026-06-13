package main

import (
	"context"
	"flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
	"vllm-gateway/internal/telemetry"

	"google.golang.org/grpc"

	"vllm-gateway/internal/balancer"
	"vllm-gateway/internal/health"
	"vllm-gateway/internal/server"
	"vllm-gateway/internal/vllm"
	pb "vllm-gateway/proto"
)

func main() {
	grpcAddr := flag.String("addr", ":50051", "gRPC 网关监听地址")
	backends := flag.String("backends", "http://localhost:8000,http://localhost:8001", "vLLM 实例地址,逗号分隔")
	model := flag.String("model", "qwen7b", "served-model-name")
	hcInterval := flag.Duration("hc-interval", 5*time.Second, "健康检查间隔")
	hcTimeout := flag.Duration("hc-timeout", 3*time.Second, "单次探活超时")
	flag.Parse()

	shutdown, err := telemetry.InitTracer(context.Background())
	if err != nil {
		log.Fatalf("初始化 tracer 失败: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	// 启动 Prometheus metrics 端点(独立端口 :2112)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("[metrics] Prometheus 端点: http://localhost:2112/metrics")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Printf("[metrics] 端点启动失败: %v", err)
		}
	}()

	urls := strings.Split(*backends, ",")
	instances := make([]*balancer.Instance, 0, len(urls))
	for i, u := range urls {
		u = strings.TrimSpace(u)
		instances = append(instances, &balancer.Instance{
			ID:     "vllm-" + itoa(i),
			Client: vllm.NewClient(u, *model),
		})
		log.Printf("注册实例 vllm-%d -> %s", i, u)
	}

	lb := balancer.New(instances)

	// 启动后台健康检查器
	checker := health.New(lb, *hcInterval, *hcTimeout)
	checker.Start(context.Background())
	log.Printf("🩺 健康检查器已启动,间隔=%v 超时=%v", *hcInterval, *hcTimeout)

	gw := server.NewGatewayServer(lb)

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
