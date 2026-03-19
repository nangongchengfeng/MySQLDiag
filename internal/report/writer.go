// Package report 提供诊断报告生成功能
//
// 该包负责收集诊断过程中的观察结果,并生成
// 详细的Markdown格式诊断报告。
package report

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Writer 报告生成器
//
// Writer 用于记录诊断过程中的每一轮观察,并最终生成
// 完整的Markdown格式报告。
type Writer struct {
	outputDir    string
	observations []Observation
	startTime    time.Time
	mu           sync.Mutex
	logger       *slog.Logger
}

// NewWriter 创建一个新的报告生成器
//
// 参数:
//   - outputDir: 报告输出目录
//
// 返回: 新的报告生成器实例,或错误信息
func NewWriter(outputDir string) (*Writer, error) {
	w := &Writer{
		outputDir:    outputDir,
		observations: []Observation{},
		startTime:    time.Now(),
		logger:       slog.Default(),
	}

	// 确保输出目录存在
	if err := w.ensureOutputDir(); err != nil {
		return nil, err
	}

	return w, nil
}

// ensureOutputDir 确保输出目录存在
func (w *Writer) ensureOutputDir() error {
	if _, err := os.Stat(w.outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(w.outputDir, 0755); err != nil {
			return fmt.Errorf("创建输出目录失败: %w", err)
		}
		w.logger.Info("已创建输出目录", "dir", w.outputDir)
	}
	return nil
}

// AddObservation 添加一条观察记录到报告
//
// 参数:
//   - tool: 工具名称
//   - action: 动作描述
//   - inputData: 输入数据
//   - observation: 观察结果
//   - roundNum: 诊断轮次
func (w *Writer) AddObservation(tool, action, inputData, observation string, roundNum int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	obs := NewObservation(roundNum, tool, action, inputData, observation)
	w.observations = append(w.observations, obs)
	w.logger.Info("观察已记录", "round", roundNum, "tool", tool, "action", action)
}

// GenerateMarkdown 生成Markdown格式的完整报告
//
// 参数:
//   - finalAnswer: 最终诊断结果
//   - metadata: 元数据(可选)
//
// 返回: 报告文件路径,或错误信息
func (w *Writer) GenerateMarkdown(finalAnswer string, metadata map[string]any) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	endTime := time.Now()
	duration := endTime.Sub(w.startTime)

	filename := generateFilename(w.startTime)
	filepath := filepath.Join(w.outputDir, filename)

	content := buildMarkdownReport(
		w.startTime,
		duration,
		w.observations,
		finalAnswer,
		metadata,
	)

	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("写入报告失败: %w", err)
	}

	w.logger.Info("报告已写入", "path", filepath)
	return filepath, nil
}

// GenerateSummaryText 生成简洁的文本摘要
//
// 参数:
//   - finalAnswer: 最终诊断结果
//
// 返回: 文本摘要
func (w *Writer) GenerateSummaryText(finalAnswer string) string {
	w.mu.Lock()
	defer w.mu.Unlock()

	return buildSummaryText(finalAnswer, w.observations)
}

// Observations 返回所有观察记录的副本
func (w *Writer) Observations() []Observation {
	w.mu.Lock()
	defer w.mu.Unlock()

	result := make([]Observation, len(w.observations))
	copy(result, w.observations)
	return result
}

// StartTime 返回诊断开始时间
func (w *Writer) StartTime() time.Time {
	return w.startTime
}

// SetLogger 设置日志记录器
func (w *Writer) SetLogger(logger *slog.Logger) {
	w.logger = logger
}
