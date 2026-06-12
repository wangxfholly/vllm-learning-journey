# Day 12 - 综合项目: Mini Agent (本地工具调用引擎)

## 做了什么
60 行代码搭出端到端本地 Agent: 自然语言 → 工具路由 → 执行 Python 函数
→ 流式自然语言回复。零外部 API，全靠自己的 7B 推理引擎。

## 三段式 ReAct 闭环
1. 路由(Reason): temperature=0 + guided_json 锁死 → 100% 合法 JSON
2. 执行(Act): 真实 Python 函数(get_weather/calculate) → 模型决策，代码做事
3. 回复(Respond): temperature=0.7 → 自然有人味

## 验收(三条路径全对)
| 输入 | 路由 | 工具结果 | 回复 |
|---|---|---|---|
| 北京天气 | get_weather, city=北京 | 晴24°C | 自然回复 ✓ |
| (135+27)*8 | calculate, expression | 1296 | 1296 ✓ |
| 你叫什么名字 | none | None | 纯对话 ✓ |
三次 json.loads 零失败。

## 整合的两周技术
- AWQ 量化加载 7B (Day07) — 单卡跑大模型
- 显存/max_model_len 配置 (Day08)
- guided decoding (Day11) — 工具参数 100% 合法
- 双 temperature (Day03) — 路由用0求稳，回复用0.7求自然
- (可扩展流式输出 Day04)

## 关键工程认知
1. Agent 本质: 模型做决策(选工具+抽参)，代码做实事(执行)。算数交给
   eval 而非让 LLM 硬算 → LLM 算数不可靠。
2. 同一链路不同环节用不同采样策略，是工程成熟度标志。
3. 日志 xgrammar fallback outlines: schema 有高级特性。追求路由低延迟
   可简化 schema 走 xgrammar。
4. 模型对自身身份记得准(Qwen=阿里)，对冷门外部事实(vLLM)就编。

## 下一步 Day 13
Week 2 复盘 + 技术博客(同 Day06 的 A+B 风格)。
