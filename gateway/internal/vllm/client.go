package vllm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client 封装对单个 vLLM 实例的 HTTP 调用
type Client struct {
	BaseURL    string // 例如 http://localhost:8000
	Model      string // served-model-name,例如 qwen7b
	httpClient *http.Client
}

func NewClient(baseURL, model string) *Client {
	return &Client{
		BaseURL: baseURL,
		Model:   model,
		// 给一个合理超时,避免请求永久挂死(后面会针对流式单独处理)
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// ---- 与 vLLM OpenAI 接口对应的请求/响应结构 ----
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float32       `json:"temperature"`
	MaxTokens   int32         `json:"max_tokens"`
	Stream      bool          `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int32 `json:"prompt_tokens"`
		CompletionTokens int32 `json:"completion_tokens"`
	} `json:"usage"`
}

// ChatResult 是给上层(gRPC server)用的结果结构,屏蔽 OpenAI 细节
type ChatResult struct {
	Content          string
	PromptTokens     int32
	CompletionTokens int32
}

// Chat 发起一次非流式请求
func (c *Client) Chat(ctx context.Context, messages []chatMessage, temperature float32, maxTokens int32) (*ChatResult, error) {
	reqBody := chatRequest{
		Model:       c.Model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Stream:      false,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := c.BaseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("构造 HTTP 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("调用 vLLM 失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vLLM 返回非 200: %d, body=%s", resp.StatusCode, string(body))
	}

	var cr chatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return nil, fmt.Errorf("解析 vLLM 响应失败: %w", err)
	}
	if len(cr.Choices) == 0 {
		return nil, fmt.Errorf("vLLM 响应无 choices")
	}

	return &ChatResult{
		Content:          cr.Choices[0].Message.Content,
		PromptTokens:     cr.Usage.PromptTokens,
		CompletionTokens: cr.Usage.CompletionTokens,
	}, nil
}

// BuildMessages 把 (role, content) 列表转成内部结构(给 server 层调用)
func BuildMessages(pairs [][2]string) []chatMessage {
	msgs := make([]chatMessage, 0, len(pairs))
	for _, p := range pairs {
		msgs = append(msgs, chatMessage{Role: p[0], Content: p[1]})
	}
	return msgs
}

// ---- Day25: 健康检查 ----

// HealthShallow 浅探活:GET /health,只看进程/端口是否活着
func (c *Client) HealthShallow(ctx context.Context) error {
	url := c.BaseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("浅探活失败(端口不通): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("浅探活非200: %d", resp.StatusCode)
	}
	return nil
}

// HealthDeep 深探活:发一个真实的短推理,验证 GPU 真的能算
// 能抓住"端口通但 GPU OOM/假死"这类浅探活漏掉的故障
func (c *Client) HealthDeep(ctx context.Context) error {
	msgs := []chatMessage{{Role: "user", Content: "hi"}}
	// max_tokens=1,最小代价验证推理链路通畅
	_, err := c.Chat(ctx, msgs, 0.0, 1)
	if err != nil {
		return fmt.Errorf("深探活失败(GPU不可用/假死): %w", err)
	}
	return nil
}

// ---- Day27: 流式 ----

// StreamChunk 流式返回的单个片段
type StreamChunk struct {
	Delta    string // 本次新增文本
	Finished bool   // 是否结束
	Err      error  // 出错(中途失败)
}

// ChatStream 发起流式请求,通过 channel 边收边吐 delta
func (c *Client) ChatStream(ctx context.Context, messages []chatMessage, temperature float32, maxTokens int32) (<-chan StreamChunk, error) {
	reqBody := chatRequest{
		Model:       c.Model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Stream:      true, // 关键:开启流式
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := c.BaseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发起流式请求失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("vLLM 流式非200: %d, %s", resp.StatusCode, string(body))
	}

	out := make(chan StreamChunk)
	go func() {
		defer close(out)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					out <- StreamChunk{Err: fmt.Errorf("读流出错(中途失败): %w", err)}
				}
				return
			}
			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				out <- StreamChunk{Finished: true}
				return
			}
			// 解析增量
			var sc struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(payload), &sc); err != nil {
				continue // 跳过无法解析的行
			}
			if len(sc.Choices) > 0 && sc.Choices[0].Delta.Content != "" {
				out <- StreamChunk{Delta: sc.Choices[0].Delta.Content}
			}
		}
	}()
	return out, nil
}
