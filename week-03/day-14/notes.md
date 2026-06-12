# Day 14:从 LLM() 到 OpenAI 兼容在线服务

## 今天干了什么
把前两周一直用的进程内嵌 LLM() 换成了常驻的 OpenAI 兼容 HTTP 服务。

启动命令:
python -m vllm.entrypoints.openai.api_server \
  --model ~/vllm-learning-journey/models/Qwen2.5-7B-Instruct-AWQ \
  --quantization awq_marlin \
  --gpu-memory-utilization 0.90 \
  --max-model-len 2048 \
  --served-model-name qwen7b \
  --port 8000

验收:curl http://localhost:8000/v1/chat/completions 成功返回标准 JSON。

## 返回 JSON 关键字段
- model = "qwen7b":served-model-name 实现了模型名与物理路径解耦,换后端不用改客户端
- finish_reason = "stop":模型输出 EOS 自然结束;若是 "length" 则是 max_tokens 截断
- usage:prompt_tokens 44 / completion_tokens 27 / total 71,这是云端计费的口径,本地服务原样实现
- tool_calls = []:原生支持 function calling 协议,Day 16 会让它非空

## 离线 vs 在线 的本质区别
- 离线 LLM():模型随脚本生死,每次实验重新 load 13GB,只有当前进程能用
- 在线 api_server:模型常驻显存,一次加载无限复用,任何 HTTP 客户端都能调,走 OpenAI 标准协议
- 生产环境永远用在线服务:模型加载是最贵操作,常驻把这个成本摊销到零

## 两终端工作流
- 终端1:跑 api_server,常驻不关
- 终端2:发 curl / 跑测试脚本
- 服务保持运行,Day 15 让 Mini Agent 直接连 localhost:8000

## 踩坑
- served-model-name 一定要显式设置,否则 model 字段会回显一长串磁盘路径,客户端写死路径就失去了解耦意义
