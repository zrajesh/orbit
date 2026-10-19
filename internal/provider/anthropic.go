// Package provider contains LLM adapters. Wire types stay private to each adapter.
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"orbit/internal/core"
)

const (
	anthropicURL     = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
	defaultMaxTokens = 4096
)

type Anthropic struct {
	apiKey string
	model  string
	client *http.Client
}

func NewAnthropic(apiKey, model string) *Anthropic {
	return &Anthropic{apiKey: apiKey, model: model, client: &http.Client{}}
}

var _ core.Provider = (*Anthropic)(nil)

type wireBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

type wireMessage struct {
	Role    string      `json:"role"`
	Content []wireBlock `json:"content"`
}

type wireTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type wireRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	System    string        `json:"system,omitempty"`
	Messages  []wireMessage `json:"messages"`
	Tools     []wireTool    `json:"tools,omitempty"`
}

type wireResponse struct {
	Content    []wireBlock `json:"content"`
	StopReason string      `json:"stop_reason"`
	Usage      struct {
		InputTokens     int `json:"input_tokens"`
		OutputTokens    int `json:"output_tokens"`
		CacheReadTokens int `json:"cache_read_input_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (a *Anthropic) Complete(ctx context.Context, req core.CompletionRequest) (core.CompletionResponse, error) {
	body, err := json.Marshal(toWire(a.model, req))
	if err != nil {
		return core.CompletionResponse{}, fmt.Errorf("anthropic: encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicURL, bytes.NewReader(body))
	if err != nil {
		return core.CompletionResponse{}, err
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return core.CompletionResponse{}, fmt.Errorf("anthropic: request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.CompletionResponse{}, fmt.Errorf("anthropic: read response: %w", err)
	}

	var wr wireResponse
	if err := json.Unmarshal(raw, &wr); err != nil {
		return core.CompletionResponse{}, fmt.Errorf("anthropic: decode response (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		msg := string(raw)
		if wr.Error != nil {
			msg = wr.Error.Message
		}
		return core.CompletionResponse{}, fmt.Errorf("anthropic: status %d: %s", resp.StatusCode, msg)
	}
	return fromWire(wr), nil
}

func toWire(model string, req core.CompletionRequest) wireRequest {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	wr := wireRequest{Model: model, MaxTokens: maxTokens, System: req.System}

	for _, m := range req.Messages {
		wm := wireMessage{Role: string(m.Role)}
		for _, b := range m.Content {
			switch b.Type {
			case core.BlockText:
				wm.Content = append(wm.Content, wireBlock{Type: "text", Text: b.Text})
			case core.BlockToolUse:
				wm.Content = append(wm.Content, wireBlock{Type: "tool_use", ID: b.ToolUseID, Name: b.Name, Input: b.Input})
			case core.BlockToolResult:
				wm.Content = append(wm.Content, wireBlock{Type: "tool_result", ToolUseID: b.ToolUseID, Content: b.Content, IsError: b.IsError})
			}
		}
		wr.Messages = append(wr.Messages, wm)
	}
	for _, t := range req.Tools {
		wr.Tools = append(wr.Tools, wireTool{Name: t.Name, Description: t.Description, InputSchema: t.InputSchema})
	}
	return wr
}

func fromWire(wr wireResponse) core.CompletionResponse {
	msg := core.Message{Role: core.RoleAssistant}
	for _, b := range wr.Content {
		switch b.Type {
		case "text":
			msg.Content = append(msg.Content, core.ContentBlock{Type: core.BlockText, Text: b.Text})
		case "tool_use":
			msg.Content = append(msg.Content, core.ContentBlock{Type: core.BlockToolUse, ToolUseID: b.ID, Name: b.Name, Input: b.Input})
		}
	}
	return core.CompletionResponse{
		Message:    msg,
		StopReason: core.StopReason(wr.StopReason),
		Usage: core.Usage{
			InputTokens:     wr.Usage.InputTokens,
			OutputTokens:    wr.Usage.OutputTokens,
			CacheReadTokens: wr.Usage.CacheReadTokens,
		},
	}
}
