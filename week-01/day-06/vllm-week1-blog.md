# 10 年后端转 AI:我用一周跑通并吃透 vLLM(含真实压测数据)

> 我是一个写了 10 年后端的工程师,做过 LangChain / LangGraph / Eino 的 Agent 应用,但一直停留在"调 API"层面。这一周我决定沉到推理引擎里,用 vLLM 从零搭一遍、压一遍、抠一遍原理。这篇是我的 Week 1 复盘——既有踩坑实录(A),也有原理科普(B)。所有数据都是我自己机器上跑出来的,不是抄文档。

机器:单卡 L4(24GB),vLLM 0.7.3,模型 Qwen2.5-1.5B-Instruct。

## 一、环境踩坑:三个让我卡了一下午的错

1. config.json not found:脚本用相对路径找模型,换目录就崩。解决:用 __file__ 算绝对路径,脚本放哪都能跑。
2. git push 报 pack exceeds maximum allowed size (2.00 GiB):我把模型权重先 commit 了,后补的 .gitignore 拦不住已入库的文件。教训:.gitignore 只对未跟踪文件生效,对已进历史的文件无效。一定要先建 .gitignore 再 git add。补救:rm -rf .git(文件保留只删历史)→ 重新 init → 确认 .gitignore 生效 → 重新提交。
3. heredoc 卡在 > 不动:内容含 markdown 三反引号代码块时 bash 被带歪一直等输入。教训:写 notes 别在 heredoc 里塞代码块,或用编辑器/base64 落地。

## 二、学会读 vLLM 的启动日志(这才是 ground truth)

很多人用 torch.cuda.memory_allocated() 量显存,结果 FP16 和 AWQ 量出来一样大——因为它量的是 vLLM 预分配的内存池(gpu_memory_utilization=0.85),不是权重本身。真正的权重显存在启动日志里:FP16 是 model weights take 2.89GiB;AWQ 是 Loading model weights took 1.1023 GB。记住:日志才是事实,torch API 会骗你。

## 三、Continuous Batching:我实测的 8.48 倍

串行(for 循环一条条 generate)vs 批量(10 条一次性 generate):

| 方式 | 吞吐 |
|------|------|
| 串行 | 71.1 tokens/s |
| 批量 | 603.1 tokens/s |
| 加速比 | 8.48x |

原理:传统 batching 要等一批里最慢那条跑完才收尾,GPU 大量空转。vLLM 用 Continuous Batching(迭代级调度):每生成一个 token 就重新调度,谁跑完谁下车、新请求随时上车,把 GPU 利用率拉满。这是它吞吐高的核心原因。

## 四、服务化:一行 base_url 就能换模型

离线 LLM() 适合跑批,线上得起服务:python -m vllm.entrypoints.openai.api_server --model <path> --port 8000。它直接暴露 OpenAI 兼容接口(/v1/models、/v1/chat/completions)。这意味着把 client 的 base_url 指到 http://localhost:8000/v1、api_key 设成 EMPTY 即可。我之前 LangChain 调 GPT 的逻辑,只改一个 base_url 就指到本地模型了,业务代码一行不动。这就是 OpenAI 兼容协议的价值:模型可替换。

## 五、采样参数:从"能跑"到"可控"

| 参数 | 作用 | 体感 |
|------|------|------|
| temperature=0 | 贪心解码,完全确定 | 同输入永远同输出 |
| top_p | 核采样,砍掉长尾低概率词 | 0.5 时回答更收敛 |
| seed | 固定随机种子,复现 | 调试/对比必备 |
| repetition_penalty | 抗复读 | 长文本设 1.3 明显不啰嗦 |

反直觉点:temperature=0 不是"质量最高",是"最确定"。要创意调高,要稳定调低,看场景。

## 六、并发压测:吞吐随并发涨,延迟也涨

aiohttp + asyncio.gather,并发档位 1/5/10/20:

| 并发 | 吞吐 | 单请求延迟 |
|------|------|-----------|
| 1 | 69.6 tokens/s | 0.88s |
| 20 | 811.1 tokens/s | 1.42s |
| 变化 | 11.6x | +0.54s |

结论:吞吐↑和单请求延迟↑是一对 trade-off。并发越高总吞吐越爽,但每个用户等得越久。生产上要按 SLA 找平衡点,不是越高越好。

## 七、流式输出:TTFT 和 TPOT 才是用户体感

stream=True 实测:TTFT(首 token 延迟)=0.292s,即用户多久看到第一个字;TPOT(每 token 延迟)=14.1ms,即出字有多快。原理:一次推理分两段——Prefill(预填充)处理你的 prompt,算力密集,决定 TTFT;Decode(解码)一个个吐 token,访存密集,决定 TPOT。所以"首字快不快"和"吐字流不流畅"是两个独立问题,优化手段也不同。流式的意义就是把 TTFT 体感拉到极致,哪怕总时长一样,用户感觉快多了。

## 八、量化:AWQ 省 62% 显存还快 2.1 倍

FP16 vs AWQ(4bit)对比:

| 指标 | FP16 | AWQ |
|------|------|-----|
| 权重显存 | 2.89 GiB | 1.10 GiB(省 62%) |
| 吞吐 | 基准 | 2.1x |

为什么 AWQ 又省又快?省:4bit 存权重,体积砍到 1/4 左右。快:vLLM 用 awq_marlin 这个专为 4bit 优化的 kernel,访存量小,decode 阶段(访存密集)直接受益。AWQ 是 activation-aware(感知激活值重要性)的量化,精度损失小,是 vLLM 生产环境首选。其他还有 GPTQ、FP8(需 Ada/Hopper,L4 支持)、GGUF(llama.cpp/Ollama 那套)。

一个真实翻车:1.5B 小模型问它 vLLM 是什么,它一本正经胡说 "Vector Language Model"(错的,v 是 PagedAttention 的 virtual memory)。小模型幻觉严重,这就是为什么要 RAG、要上更大的模型。

## 九、Week 1 小结 & 下一步

这一周我把一条链路打通了:原理(prefill/decode)→ 指标(TTFT/TPOT/吞吐)→ 实测数据,概念闭环了。核心收获:vLLM 高吞吐的根 = PagedAttention + Continuous Batching;服务化靠 OpenAI 兼容协议,模型可热插拔;量化是单卡跑大模型的钥匙。

下一步(Week 2):用量化在单卡上跑 7B/14B;投机解码(speculative decoding);prefill/decode 分离部署。

写了 10 年后端,第一次觉得离"模型怎么真正跑起来"这么近。继续。
