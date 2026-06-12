# Day 11 - 结构化输出 + Guided Decoding

## 核心问题
传统"prompt 求模型吐 JSON"是祈求不是保证: 偶发吐 markdown 包裹/客套话/
多余字符 → json.loads 崩 → Agent 流程挂。温度越高越容易翻车。

## Guided Decoding 原理(质变)
在采样每一步用状态机(基于 JSON Schema/正则/选项)强行屏蔽所有"不合法"token,
不合法 token 概率直接置零，模型根本没机会吐错。
→ 不是"求模型听话"，是数学上锁死输出空间，100% 符合 schema。

## vLLM 四种引导模式
- guided_json: 符合 JSON Schema(工具调用参数、抽取)
- guided_choice: 必须是给定选项之一(分类、意图)
- guided_regex: 符合正则(电话/日期/ID)
- guided_grammar: 符合 EBNF(SQL/DSL)

## 实验: 智能家居意图抽取(7B AWQ, guided_json)
| 输入 | 输出 | 点评 |
|---|---|---|
| 空调调到26度 | {control_device, air_conditioner, increase, 26} | 满分 |
| 关卧室灯 | {control_device, 灯, off, 0} | device自由字段不稳定(中英混) |
| 今天天气 | {query_weather, smart_home_assistant, none, 0} | intent对,device瞎编 |
三条全部 json.loads 成功。

## 关键认知
1. guided decoding 保证"格式合法"，不保证"语义完美"。格式是硬约束，
   语义靠模型能力 + schema 设计。
2. 自由 string 字段不稳定，要对接固定值就设成 enum 或后处理映射。
3. 日志: xgrammar(快，不支持 pattern/数字范围) 自动 fallback outlines(全，慢)。
   schema 加高级约束会牺牲速度，能简单就别堆复杂。

## 对 Agent 的价值
别再在 prompt 里跪求模型听话。推理层用 guided decoding 锁死输出，是把
Agent 从"demo能跑"到"生产可靠"的关键。LangChain/Eino structured output
底层就靠这个。

## 下一步 Day 12
综合项目: 把两周学的整合成一个能用的推理服务 demo。
