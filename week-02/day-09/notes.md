# Day 09 - 投机解码 Speculative Decoding

## 核心思想
decode 阶段是访存密集型(瓶颈是搬权重，算力闲置)。投机解码利用闲置算力:
小模型/历史文本"猜"出未来多个 token(草稿)，大模型一次前向并行验证。
猜对就白赚多个 token，猜错也几乎不亏。最终由大模型拍板 → 无损加速。

## 实验结果(7B AWQ, temperature=0, 同样 2 道题)
| 模式 | 输出 tokens | 耗时 | 吞吐 |
|------|------|------|------|
| baseline 纯7B | 1011 | 10.30s | 98.14 t/s |
| n-gram 投机解码 | 1011 | 9.59s | 105.46 t/s (+7.5%) |
输出 token 数完全相同 → 证明无损。

## 踩坑:草稿模型方案失败
用 1.5B 当 7B 的草稿模型报错:
  AssertionError: vocab_sizes[0] == vocab_size (spec_decode_worker.py:1260)
原因: 投机解码要求草稿与目标模型 vocab_size 完全一致。Qwen2.5-7B 和 1.5B
虽同系列，但 embedding 矩阵 padding 到不同对齐边界，vocab_size 不等。
vLLM 0.7.3 对异构模型组合断言卡得很死。
→ 解法: 改用 [ngram] 模式，无需第二个模型。

## n-gram 投机解码(本次用的方案)
- 不用草稿模型，从已生成文本里找重复片段当草稿
- 零额外显存(KV Cache 13.07G、并发 59.75x，和纯 7B 几乎一样)
- 参数: speculative_model="[ngram]", num_speculative_tokens=5,
        ngram_prompt_lookup_max=4

## 加速幅度取决于"可预测性"
- 代码/JSON/模板/长文档续写(高重复) → 1.5x~2x
- 发散创意文本(解释/创作) → 几乎不加速
本次两道题混合(含发散的 PagedAttention 解释)→ 被拖累，整体仅 +7.5%

## 对 Agent 的价值
Agent 大量是结构化输出(JSON 工具调用、固定格式) → 投机解码黄金场景。
开 n-gram 可降工具调用延迟 30%~50%，无损零风险，性价比极高。

## 下一步 Day 10
多并发服务调优: max_num_seqs / 调度参数 / 显存与吞吐的权衡。
