// Package core holds orbit's provider-neutral domain types.
package core

import "encoding/json"

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type StopReason string

const (
	StopEndTurn   StopReason = "end_turn"
	StopToolUse   StopReason = "tool_use"
	StopMaxTokens StopReason = "max_tokens"
)

type BlockType string

const (
	BlockText       BlockType = "text"
	BlockToolUse    BlockType = "tool_use"
	BlockToolResult BlockType = "tool_result"
)

// ContentBlock is a tagged union; Type decides which fields are meaningful.
type ContentBlock struct {
	Type BlockType

	Text string // BlockText

	ToolUseID string          // BlockToolUse (the id) and BlockToolResult (the id it answers)
	Name      string          // BlockToolUse
	Input     json.RawMessage // BlockToolUse

	Content string // BlockToolResult
	IsError bool   // BlockToolResult
}

type Message struct {
	Role    Role
	Content []ContentBlock
}

func TextMessage(role Role, text string) Message {
	return Message{Role: role, Content: []ContentBlock{{Type: BlockText, Text: text}}}
}

type ToolDefinition struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

type Usage struct {
	InputTokens     int
	OutputTokens    int
	CacheReadTokens int
}

func (u *Usage) Add(o Usage) {
	u.InputTokens += o.InputTokens
	u.OutputTokens += o.OutputTokens
	u.CacheReadTokens += o.CacheReadTokens
}

// CompletionRequest carries the full history on every call; providers are stateless.
type CompletionRequest struct {
	System    string
	Messages  []Message
	Tools     []ToolDefinition
	MaxTokens int
}

type CompletionResponse struct {
	Message    Message
	StopReason StopReason
	Usage      Usage
}

// ToolUses returns the tool_use blocks of a message in order.
func (m Message) ToolUses() []ContentBlock {
	var out []ContentBlock
	for _, b := range m.Content {
		if b.Type == BlockToolUse {
			out = append(out, b)
		}
	}
	return out
}

// Text concatenates the text blocks of a message.
func (m Message) Text() string {
	var s string
	for _, b := range m.Content {
		if b.Type == BlockText {
			s += b.Text
		}
	}
	return s
}
