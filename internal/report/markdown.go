package report

import (
	"fmt"
	"strings"
	"time"
)

// formatCodeBlock 将内容格式化为Markdown代码块
//
// 如果内容超过5000字符,会被截断。
func formatCodeBlock(content string, language string) string {
	if content == "" {
		return ""
	}
	if len(content) > 5000 {
		content = content[:5000] + "\n... [已截断]"
	}
	return fmt.Sprintf("```%s\n%s\n```", language, content)
}

// generateFilename 生成唯一的报告文件名
func generateFilename(startTime time.Time) string {
	timestamp := startTime.Format("20060102_150405")
	return fmt.Sprintf("diagnostic_report_%s.md", timestamp)
}

// buildMarkdownReport 构建完整的Markdown报告
func buildMarkdownReport(
	startTime time.Time,
	duration time.Duration,
	observations []Observation,
	finalAnswer string,
	metadata map[string]any,
) string {
	var lines []string

	lines = append(lines, "# MySQL/Linux 诊断报告")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("- **生成时间**: %s", startTime.Format("2006-01-02 15:04:05")))
	lines = append(lines, fmt.Sprintf("- **总耗时**: %.2f 秒", duration.Seconds()))
	lines = append(lines, fmt.Sprintf("- **诊断轮次**: %d", getMaxRound(observations)))
	lines = append(lines, "")

	// 添加元数据
	if metadata != nil && len(metadata) > 0 {
		lines = append(lines, "## 配置信息")
		lines = append(lines, "")
		for key, value := range metadata {
			lines = append(lines, fmt.Sprintf("- **%s**: %v", key, value))
		}
		lines = append(lines, "")
	}

	// 添加最终诊断
	lines = append(lines, "## 最终诊断")
	lines = append(lines, "")
	lines = append(lines, finalAnswer)
	lines = append(lines, "")

	// 添加诊断过程
	lines = append(lines, "## 诊断过程")
	lines = append(lines, "")

	if len(observations) == 0 {
		lines = append(lines, "*没有记录观察数据。*")
		lines = append(lines, "")
	} else {
		// 按轮次分组
		rounds := make(map[int][]Observation)
		for _, obs := range observations {
			rounds[obs.Round] = append(rounds[obs.Round], obs)
		}

		// 按轮次顺序输出
		for roundNum := 1; roundNum <= getMaxRound(observations); roundNum++ {
			roundObs, ok := rounds[roundNum]
			if !ok {
				continue
			}

			lines = append(lines, fmt.Sprintf("### 第 %d 轮", roundNum))
			lines = append(lines, "")

			for _, obs := range roundObs {
				lines = append(lines, fmt.Sprintf("**工具**: `%s.%s`", obs.Tool, obs.Action))
				lines = append(lines, "")
				lines = append(lines, "**输入**:")
				lines = append(lines, "")
				lines = append(lines, formatCodeBlock(obs.Input, ""))
				lines = append(lines, "")
				lines = append(lines, "**观察结果**:")
				lines = append(lines, "")
				lines = append(lines, formatCodeBlock(strings.TrimSpace(obs.Observation), ""))
				lines = append(lines, "")
			}
		}
	}

	lines = append(lines, "---")
	lines = append(lines, "")
	lines = append(lines, "*由 MySQL 诊断 Agent 自动生成*")

	return strings.Join(lines, "\n")
}

// buildSummaryText 构建简洁的文本摘要
func buildSummaryText(finalAnswer string, observations []Observation) string {
	var lines []string

	lines = append(lines, strings.Repeat("=", 60))
	lines = append(lines, "        MySQL/Linux 诊断报告摘要")
	lines = append(lines, strings.Repeat("=", 60))
	lines = append(lines, "")
	lines = append(lines, "最终诊断:")
	lines = append(lines, strings.Repeat("-", 40))
	lines = append(lines, finalAnswer)
	lines = append(lines, "")
	lines = append(lines, strings.Repeat("=", 60))
	lines = append(lines, fmt.Sprintf("总诊断轮次: %d", getMaxRound(observations)))

	return strings.Join(lines, "\n")
}

// getMaxRound 获取最大的轮次数
func getMaxRound(observations []Observation) int {
	maxRound := 0
	for _, obs := range observations {
		if obs.Round > maxRound {
			maxRound = obs.Round
		}
	}
	return maxRound
}
