# Day 24 — L3 起步:多实例 + 负载均衡(Round-Robin)

## 做了什么
1. internal/balancer/balancer.go:实例池 + Round-Robin 选择器
   - atomic.AddUint64 做轮询计数器,并发安全(避免高并发串号 bug)
   - 预留 Size(),为 Day25 健康检查铺路
2. server.go 改造:从持有单 client → 持有 balancer,每次请求 Pick() 选实例
3. main.go 改造:--backends 逗号分隔多后端,构建实例池(vllm-0/vllm-1)
4. 实测:连发 6 次,instance_id 严格交替 vllm-0/vllm-1,LB 生效

## 单卡跑多实例的方法
方案A(本次):两个逻辑实例指向同一后端 8000,LB 逻辑100%真实,只是后端碰巧同一个
方案B(有多卡再用):gpu-mem-util 0.45 各起一个真实例 8000/8001

## LB 策略认知(面试高频)
- Round-Robin:简单,但 LLM 请求耗时差异大(生成长度不同),易堆积
- Least-Connection:按当前活跃连接,较好
- Least-Pending/队列深度:对应 vLLM num_requests_waiting,最贴合 LLM,最优
- 关键:普通Web请求耗时均匀轮询够用;LLM耗时天差地别,按负载路由远好于轮询

## 下一步
Day 25:健康检查(浅探活/health + 深探活真实推理)+ 服务发现摘除
