# vLLM 进阶一周:单卡跑 7B、抠 KV Cache、做生产级 Agent(全程真实数据)

> Week 1 我把 vLLM 跑通了,这是 Week 2 的复盘。如果说第一周是"会用",第二周就是"会调、会救、会做产品"。我用一张 24GB 的 L4,跑起 7B、压测调优、最后搭出一个能用的本地 Agent。还是 A+B 风格:踩坑实录 + 原理科普,数据全是自己机器上跑的。

机器:单卡 L4(24GB),vLLM 0.7.3,模型 Qwen2.5-7B-Instruct-AWQ + 1.5B。

## 一、单卡跑 7B:量化不是省钱,是换产能

7B 用 FP16 要 14GB 权重,24GB 卡很紧张。换成 AWQ 4bit,权重只占 5.2GB。但真正的认知是看启动日志算出来的显存账:

| 项目 | 占用 | 占比 |
|------|------|------|
| 7B AWQ 权重 | 5.20 GiB | 24% |
| PyTorch 激活峰值 | 1.40 GiB | 6% |
| KV Cache | 13.09 GiB | 60%+ |

重点:权重只占 24%,KV Cache 才是大头(60%)。所以量化省下的 9GB 全进了 KV Cache 池子,换来更高并发。一句话:量化的本质是把省下的显存换成吞吐和并发,不只是省钱。

## 二、KV Cache 深挖:长上下文是"双重打击"

KV Cache 按 block 管理(默认 16 token/block,这是 PagedAttention 借鉴 OS 分页内存的设计)。我扫了不同 max_model_len 看并发上限:

| max_model_len | 激活峰值 | KV Cache | 并发数 |
|------|------|------|------|
| 2048 | 1.40 GiB | 13.10 GiB | 119.81x |
| 8192 | 中间 | 13.06 GiB | 29.86x |
| 32768 | 5.51 GiB | 8.99 GiB | 5.14x |

铁律:并发数 = KV缓存池 / 单请求长度。但 32768 这组暴露了一个隐藏陷阱:长上下文不只线性吃 KV Cache,还把 PyTorch 激活峰值从 1.4G 抬到 5.5G,反过来挤占 KV Cache 池本身。结果并发从 120 跌到 5(24倍暴跌),远超简单的 16 倍反比。

教训:生产里 max_model_len 必须按实际需要设,无脑设满代价极大。

长文本 OOM 三种救法:调大 gpu_memory_utilization(扩池)、调小 max_model_len(最该优先)、enforce_eager=True(关 CUDA Graph 省显存换速度)。

## 三、投机解码:无损加速,但要踩对场景

decode 阶段是访存密集型(瓶颈是搬权重,算力闲置)。投机解码就是利用闲置算力:小模型/历史文本猜出未来多个 token,大模型一次并行验证,猜对白赚,最终由大模型拍板所以无损。

我先试了拿 1.5B 当 7B 的草稿模型,直接报错:

AssertionError: vocab_sizes[0] == vocab_size

原因:投机解码要求草稿和目标模型 vocab_size 完全一致。Qwen2.5 的 7B 和 1.5B 虽同系列,但 embedding 矩阵 padding 到不同对齐边界,vocab_size 不等,vLLM 0.7.3 断言卡死。改用 n-gram 模式(不需要第二个模型,从历史文本找草稿,零额外显存)后跑通:

| 模式 | 输出 tokens | 耗时 | 吞吐 |
|------|------|------|------|
| 纯 7B | 1011 | 10.30s | 98.14 t/s |
| n-gram 投机 | 1011 | 9.59s | 105.46 t/s |

输出 token 数完全相同 = 无损。只快 7.5% 是因为测试题里有发散文本拖后腿;换成代码/JSON 这类高重复内容能到 1.5x~2x。对做 Agent 的人:结构化输出是投机解码的黄金场景。

## 四、并发调优:max_num_seqs 是吞吐/延迟的滑动旋钮

固定灌 64 个请求,只改 max_num_seqs:

| max_num_seqs | 总耗时 | 吞吐 | 相对提升 |
|------|------|------|------|
| 4 | 64.29s | 187 t/s | 1x |
| 16 | 19.03s | 645 t/s | 3.4x |
| 64 | 8.42s | 1466 t/s | 7.8x |

两个洞察:
1. 边际收益递减。参数翻4倍,吞吐涨不到4倍(4→16 涨3.4x,16→64 涨2.3x)。因为 decode 从大 batch 获益大,但 prefill 本就打满算力,batch 再大也榨不出更多。
2. 曲线会封顶。继续加会撞 KV Cache 上限或算力饱和。甜点区在"吞吐快封顶、延迟还没爆炸"的拐点。

调优框架:离线批处理设大(只看吞吐);在线高并发设中(压测找拐点,守 P99);低延迟交互/Agent 设小(优先单请求响应)。本质和后端调线程池一样:找平衡,不是拉满。

## 五、Guided Decoding:Agent 生产化的关键拼图

传统"prompt 求模型吐 JSON"是祈求不是保证,偶发吐脏数据导致 json.loads 崩。Guided Decoding 是质变:在采样每一步用状态机屏蔽所有不合法 token,从数学上锁死输出空间,100% 符合 schema。

我做了智能家居意图抽取,三条全部 json.loads 成功。关键认知:guided decoding 保证格式合法,不保证语义完美(有一条把无关字段瞎填了)。格式是硬约束,语义靠模型能力+schema 设计。

日志里还看到 xgrammar(快,不支持高级特性)自动 fallback 到 outlines(全,慢)的引擎切换——schema 加复杂约束会牺牲速度,能简单就别堆复杂。

## 六、综合项目:60 行搭一个本地 Agent

把上面全部技术整合,做了个端到端的本地工具调用引擎:自然语言 → guided decoding 路由选工具+抽参 → 执行真实 Python 函数 → 流式自然语言回复。零外部 API。

三段式 ReAct 闭环:
- 路由(Reason): temperature=0 + guided_json,要稳要合法
- 执行(Act): 真实 Python 函数,模型做决策代码做实事
- 回复(Respond): temperature=0.7,要自然

实测三条 case(查天气/算数/闲聊)全对,json.loads 零失败。最有意思的是算数题:模型不自己算(LLM 算数不可靠),而是抽出表达式交给 Python eval。这就是 Agent 的本质——模型负责理解和决策,工具负责精确执行。

同一条链路不同环节用不同 temperature(路由0、回复0.7),是工程成熟度的标志。

## 七、Week 2 小结:从"会用"到"会做产品"

两周下来,我完成了从"调 API 的后端"到"懂推理引擎的 AI Backend"的跨越:
- Week 1: 跑通 vLLM,理解 PagedAttention / Continuous Batching / 流式 / 量化
- Week 2: 单卡跑 7B、抠透 KV Cache 显存账、投机解码、并发调优、guided decoding、做出能用的 Agent

最大的认知升级:推理引擎不是黑盒。每一个性能数字背后都有可解释的原因(显存怎么分、batch 怎么调度、token 怎么约束),而这些恰恰是上层 LangChain/Eino 框架封装掉、但出问题时必须懂的底层。

下一步: 多卡张量并行跑更大模型、prefill/decode 分离部署、把这套 Agent 接入真实业务。继续。
