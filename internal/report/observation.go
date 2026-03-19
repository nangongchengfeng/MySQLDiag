package report

import "time"

// Observation 表示一条观察记录
//
// Observation 记录诊断过程中每一轮使用的工具、执行的动作、
// 输入数据和观察结果。
type Observation struct {
	Round       int       // 诊断轮次
	Tool        string    // 工具名称
	Action      string    // 动作描述
	Input       string    // 输入数据
	Observation string    // 观察结果
	Timestamp   time.Time // 记录时间
}

// NewObservation 创建一条新的观察记录
func NewObservation(round int, tool, action, input, observation string) Observation {
	return Observation{
		Round:       round,
		Tool:        tool,
		Action:      action,
		Input:       input,
		Observation: observation,
		Timestamp:   time.Now(),
	}
}
