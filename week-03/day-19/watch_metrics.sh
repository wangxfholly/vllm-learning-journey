#!/bin/bash
# 每 0.5 秒拉一次关键指标,实时观察系统状态
for i in $(seq 1 40); do
  curl -s http://localhost:8000/metrics \
    | grep -E "num_requests_running|num_requests_waiting|gpu_cache_usage_perc|gpu_prefix_cache_hit_rate|num_preemptions_total" \
    | grep "^vllm:" \
    | awk '{printf "%-50s %s\n", $1, $2}'
  echo "------- $(date +%T) -------"
  sleep 0.5
done
