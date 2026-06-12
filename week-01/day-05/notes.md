# Day 05 — 量化 Quantization

## 为什么量化
FP16 下 7B≈14G、14B≈28G、72B≈144G,小卡装不下。量化把权重 16bit 压到 4/8bit,显存砍半到1/4,让大模型塞进小卡。

## 主流方案
- AWQ:activation-aware,保护重要权重,vLLM 生产首选
- GPTQ:二阶梯度逐层量化,生态成熟
- FP8:8bit浮点,需 Hopper/Ada(L4支持),近乎无损
- GGUF(Q4_K_M):llama.cpp/Ollama 用,CPU/Mac 友好

## 实测对比(Qwen2.5-1.5B FP16 vs AWQ)
| 指标 | FP16 | AWQ 4bit |
|---|---|---|
| 权重显存 | 2.89 GB | 1.10 GB(省62%) |
| 速度 | 71.2 tok/s | 150.3 tok/s(快2.1x) |
| KV Cache | 14.34 GB | 16.12 GB |
| 最大并发 | 262x | 294x |
| 质量 | 正确 | 正确(几乎无损) |

## 两个重要教训
1. 测显存别用 torch.cuda.memory_allocated()(测的是vLLM显存池,不准)。要看启动日志 "model weights take X GiB"——日志才是 ground truth。
2. 小模型AWQ反而快2倍:日志显示 Using awq_marlin kernel。marlin是4bit专用高性能算子;decode是memory-bound,权重数据量少4倍→搬得快→吐字快。量化省显存+提速双赢。

## 诚实提醒
质量"几乎无损"在简单任务成立,复杂推理/数学/长文会有可测精度损失。生产上线前必须用评测集benchmark对比,不能只看一题答对。

## Week 1 已完成
Day1跑通/ Day2批量+server / Day3采样参数 / Day4并发+流式 / Day5量化

## 明日(Day 06)
- Week 1 复盘 + 整理博客草稿
- 或:开始 7B 模型实战(用量化把更大模型跑起来)
