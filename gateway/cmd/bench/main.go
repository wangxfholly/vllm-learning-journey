package main

import (
	"context"
	"flag"
	"log"
	"sync"
	"sync/atomic"
	"time"

	pb "vllm-gateway/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "localhost:50051", "网关地址")
	concurrency := flag.Int("c", 4, "并发 goroutine 数")
	dur := flag.Duration("d", 60*time.Second, "压测时长")
	q := flag.String("q", "用一句话解释什么是负载均衡", "问题")
	flag.Parse()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()
	client := pb.NewLLMGatewayClient(conn)

	var ok, fail, degraded uint64
	var totalMs uint64
	stop := make(chan struct{})

	// 压测 worker
	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				t0 := time.Now()
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				resp, err := client.Chat(ctx, &pb.ChatRequest{
					Messages:  []*pb.Message{{Role: "user", Content: *q}},
					MaxTokens: 64,
				})
				cancel()
				ms := uint64(time.Since(t0).Milliseconds())
				if err != nil {
					atomic.AddUint64(&fail, 1)
				} else if resp.InstanceId == "degraded" {
					atomic.AddUint64(&degraded, 1)
					atomic.AddUint64(&totalMs, ms)
				} else {
					atomic.AddUint64(&ok, 1)
					atomic.AddUint64(&totalMs, ms)
				}
			}
		}()
	}

	// 每秒打印一次快照
	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		var lastTotal uint64
		for {
			select {
			case <-stop:
				return
			case <-tick.C:
				o := atomic.LoadUint64(&ok)
				f := atomic.LoadUint64(&fail)
				d := atomic.LoadUint64(&degraded)
				tot := o + f + d
				qps := tot - lastTotal
				lastTotal = tot
				avg := uint64(0)
				if o+d > 0 {
					avg = atomic.LoadUint64(&totalMs) / (o + d)
				}
				log.Printf("[bench] QPS=%d 累计: ✅成功=%d ⚠️降级=%d ❌失败=%d 平均耗时=%dms",
					qps, o, d, f, avg)
			}
		}
	}()

	log.Printf("[bench] 开始压测: 并发=%d 时长=%v 目标=%s", *concurrency, *dur, *addr)
	time.Sleep(*dur)
	close(stop)
	wg.Wait()
	log.Printf("[bench] 结束. 总成功=%d 降级=%d 失败=%d",
		atomic.LoadUint64(&ok), atomic.LoadUint64(&degraded), atomic.LoadUint64(&fail))
}
