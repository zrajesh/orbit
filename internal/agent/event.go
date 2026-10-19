package agent

import "orbit/internal/core"

type EventType string

const (
	EventText       EventType = "text"
	EventToolCall   EventType = "tool_call"
	EventToolResult EventType = "tool_result"
	EventRunEnd     EventType = "run_end"
	EventError      EventType = "error"
)

// Event is what a session emits; any frontend (CLI, SSE server, app) can consume it.
type Event struct {
	Type       EventType
	Text       string
	ToolName   string
	ToolInput  string
	ToolError  bool
	Err        error
	StopReason core.StopReason
	Usage      core.Usage // cumulative for the run, set on RunEnd and Error
}
