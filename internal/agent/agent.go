// Package agent 提供核心诊断Agent功能
//
// 该包包含诊断Agent的核心编排逻辑,使用LLM自主决定
// 需要收集哪些信息,何时可以得出诊断结论。
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// 定义接口,便于测试
type SSHTool interface {
	Execute(command string, timeout int) (string, error)
}

type MySQLTool interface {
	Query(sql string, params ...any) ([]map[string]any, error)
}

type ReportWriter interface {
	AddObservation(tool, action, inputData, observation string, roundNum int)
	GenerateMarkdown(finalAnswer string, metadata map[string]any) (string, error)
	GenerateSummaryText(finalAnswer string) string
}

type ToolCall interface {
	IsFinalAnswer() bool
	IsMySQLQuery() bool
	IsSSHExec() bool
	GetTool() string
	GetAction() string
	GetInput() string
}

// Message 表示一条对话消息
type Message struct {
	Role    string // 角色: system, user, assistant
	Content string // 消息内容
}

// Agent 诊断Agent
//
// Agent 是诊断系统的核心,负责:
// 1. 与LLM交互,获取工具调用决策
// 2. 执行工具调用
// 3. 收集观察结果
// 4. 生成最终诊断
type Agent struct {
	sshTool           SSHTool
	mysqlTool         MySQLTool
	reportWriter      ReportWriter
	apiKey            string
	baseURL           string
	model             string
	slowQueryThreshold float64
	messages          []Message
	currentRound      int
	mu                sync.Mutex
	logger            *slog.Logger
}

const (
	// MaxRounds 最大诊断轮次
	MaxRounds = 30
)

// NewAgent 创建一个新的诊断Agent
//
// 参数:
//   - sshTool: SSH工具实例
//   - mysqlTool: MySQL工具实例
//   - reportWriter: 报告生成器实例
//   - apiKey: API密钥
//   - baseURL: API基础URL
//   - model: 使用的模型名称
//   - slowQueryThreshold: 慢查询阈值(秒)
//
// 返回: 新的Agent实例
func NewAgent(
	sshTool SSHTool,
	mysqlTool MySQLTool,
	reportWriter ReportWriter,
	apiKey, baseURL, model string,
	slowQueryThreshold float64,
) *Agent {
	return &Agent{
		sshTool:           sshTool,
		mysqlTool:         mysqlTool,
		reportWriter:      reportWriter,
		apiKey:            apiKey,
		baseURL:           baseURL,
		model:             model,
		slowQueryThreshold: slowQueryThreshold,
		messages:          []Message{},
		currentRound:      0,
		logger:            slog.Default(),
	}
}

// Run 运行完整的诊断流程
//
// 参数:
//   - ctx: 上下文,用于取消和超时控制
//
// 返回: 最终诊断结果,或错误信息
func (a *Agent) Run(ctx context.Context) (string, error) {
	a.logger.Info("诊断Agent启动...")
	a.mu.Lock()
	a.currentRound = 0
	a.mu.Unlock()

	userMessage := a.buildInitialContext()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		a.mu.Lock()
		if a.currentRound >= MaxRounds {
			a.mu.Unlock()
			finalAnswer := "诊断已达到最大轮次限制仍未结束。请查看已收集的观察数据进行手动分析。"
			a.logger.Warn("已达到最大诊断轮次")
			return finalAnswer, nil
		}
		a.currentRound++
		round := a.currentRound
		a.mu.Unlock()

		a.logger.Info("--- 诊断轮次 ---", "round", round)

		// 调用LLM获取工具决策
		toolCall, err := a.callLLM(ctx, userMessage)
		if err != nil {
			return "", fmt.Errorf("调用LLM失败: %w", err)
		}

		a.logger.Info("LLM决策",
			"tool", toolCall.GetTool(),
			"action", toolCall.GetAction())

		// 检查是否是最终答案
		if toolCall.IsFinalAnswer() {
			finalAnswer := toolCall.GetInput()
			a.reportWriter.AddObservation(
				"agent",
				"final_answer",
				"",
				finalAnswer,
				round,
			)
			a.logger.Info("已得出最终诊断")
			return finalAnswer, nil
		}

		// 执行工具调用
		observation, err := a.executeTool(ctx, toolCall)
		if err != nil {
			observation = fmt.Sprintf("错误: %v", err)
			a.logger.Error("工具执行失败", "error", err)
		}

		// 记录观察
		a.reportWriter.AddObservation(
			toolCall.GetTool(),
			toolCall.GetAction(),
			toolCall.GetInput(),
			observation,
			round,
		)

		// 构建下一轮的用户消息
		userMessage = fmt.Sprintf(`来自 %s.%s 的观察结果:

【输入】
%s

【结果】
%s

接下来你想做什么?如果已经收集了足够的信息,请使用 final_answer 给出诊断结论。
`,
			toolCall.GetTool(),
			toolCall.GetAction(),
			toolCall.GetInput(),
			observation,
		)
	}
}

// buildInitialContext 构建初始上下文
func (a *Agent) buildInitialContext() string {
	return fmt.Sprintf(`MySQL/Linux 诊断会话开始

【配置信息】
- 慢查询阈值: %v 秒
- 目标: MySQL 和 Linux 系统健康检查

请开始诊断过程。建议按以下顺序进行:
1. 检查慢查询统计信息
2. 检查 MySQL 状态和配置变量
3. 检查系统资源使用情况

请确保收集足够全面的信息后再给出最终诊断。
`,
		a.slowQueryThreshold,
	)
}

// SetLogger 设置日志记录器
func (a *Agent) SetLogger(logger *slog.Logger) {
	a.logger = logger
}

// CurrentRound 返回当前诊断轮次
func (a *Agent) CurrentRound() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentRound
}

// Messages 返回消息历史的副本
func (a *Agent) Messages() []Message {
	a.mu.Lock()
	defer a.mu.Unlock()

	result := make([]Message, len(a.messages))
	copy(result, a.messages)
	return result
}
