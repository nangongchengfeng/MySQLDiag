package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LLMRequest LLM API请求
type LLMRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

// LLMResponse LLM API响应
type LLMResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// ToolCallImpl 工具调用实现
type ToolCallImpl struct {
	Tool   string `json:"tool"`
	Action string `json:"action"`
	Input  string `json:"input"`
}

func (tc *ToolCallImpl) IsFinalAnswer() bool {
	return tc.Tool == "final_answer"
}

func (tc *ToolCallImpl) IsMySQLQuery() bool {
	return tc.Tool == "mysql_query"
}

func (tc *ToolCallImpl) IsSSHExec() bool {
	return tc.Tool == "ssh_exec"
}

func (tc *ToolCallImpl) GetTool() string {
	return tc.Tool
}

func (tc *ToolCallImpl) GetAction() string {
	return tc.Action
}

func (tc *ToolCallImpl) GetInput() string {
	return tc.Input
}

// callLLM 调用LLM获取工具决策
func (a *Agent) callLLM(ctx context.Context, userMessage string) (*ToolCallImpl, error) {
	a.mu.Lock()
	a.messages = append(a.messages, Message{Role: "user", Content: userMessage})

	// 构建消息列表
	msgs := []Message{
		{Role: "system", Content: SystemPrompt},
	}
	msgs = append(msgs, a.messages...)
	a.mu.Unlock()

	req := LLMRequest{
		Model:       a.model,
		Messages:    msgs,
		Temperature: 0.3,
		MaxTokens:   3000,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := strings.TrimRight(a.baseURL, "/") + "/chat/completions"
	a.logger.Debug("调用LLM API", "url", url)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return a.handleLLMError(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return a.handleLLMError(fmt.Errorf("读取响应失败: %w", err))
	}

	if resp.StatusCode != http.StatusOK {
		return a.handleLLMError(fmt.Errorf("API返回错误状态: %d, body: %s", resp.StatusCode, string(respBody)))
	}

	var llmResp LLMResponse
	if err := json.Unmarshal(respBody, &llmResp); err != nil {
		return a.handleLLMError(fmt.Errorf("解析响应失败: %w", err))
	}

	if len(llmResp.Choices) == 0 {
		return a.handleLLMError(fmt.Errorf("API返回空响应"))
	}

	assistantMessage := llmResp.Choices[0].Message.Content

	a.mu.Lock()
	a.messages = append(a.messages, Message{Role: "assistant", Content: assistantMessage})
	a.mu.Unlock()

	// 清理JSON
	assistantMessage = strings.TrimSpace(assistantMessage)
	if strings.HasPrefix(assistantMessage, "```json") {
		assistantMessage = assistantMessage[7:]
	}
	if strings.HasPrefix(assistantMessage, "```") {
		assistantMessage = assistantMessage[3:]
	}
	if strings.HasSuffix(assistantMessage, "```") {
		assistantMessage = assistantMessage[:len(assistantMessage)-3]
	}
	assistantMessage = strings.TrimSpace(assistantMessage)

	var toolCall ToolCallImpl
	if err := json.Unmarshal([]byte(assistantMessage), &toolCall); err != nil {
		return a.handleLLMError(fmt.Errorf("解析工具调用失败: %w", err))
	}

	return &toolCall, nil
}

// handleLLMError 处理LLM错误,返回final_answer
func (a *Agent) handleLLMError(err error) (*ToolCallImpl, error) {
	a.logger.Error("LLM调用失败", "error", err)
	return &ToolCallImpl{
		Tool:   "final_answer",
		Action: "结束诊断",
		Input:  fmt.Sprintf("诊断因错误中断: %v\n\n请查看报告中的观察数据进行手动分析。", err),
	}, nil
}
