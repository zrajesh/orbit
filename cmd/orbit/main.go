package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"orbit/internal/agent"
	"orbit/internal/provider"
	"orbit/internal/tools"
)

const systemPrompt = "You are orbit, a careful coding assistant. Use the available tools to inspect files in the workspace before answering questions about them."

func main() {
	os.Exit(run())
}

func run() int {
	model := flag.String("model", envOr("ORBIT_MODEL", "claude-haiku-5-5"), "model id")
	root := flag.String("root", ".", "workspace root the file tools are confined to")
	maxTurns := flag.Int("max-turns", 20, "maximum LLM turns per run")
	maxTokens := flag.Int("max-tokens", 200_000, "maximum total tokens per run")
	flag.Parse()

	prompt := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if prompt == "" {
		fmt.Fprintln(os.Stderr, "usage: orbit [flags] <prompt>")
		return 2
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY is not set")
		return 2
	}

	sb, err := tools.NewSandbox(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sandbox:", err)
		return 1
	}
	reg := tools.NewRegistry()
	reg.Register(tools.NewReadFile(sb))
	reg.Register(tools.NewListDir(sb))

	session := agent.NewSession(agent.Config{
		Provider: provider.NewAnthropic(apiKey, *model),
		Tools:    reg,
		System:   systemPrompt,
		Budget:   agent.Budget{MaxTurns: *maxTurns, MaxTokens: *maxTokens, MaxTime: 5 * time.Minute},
	})

	// Ctrl+C cancels the context, which aborts the in-flight HTTP call and the loop.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	code := 0
	for ev := range session.Run(ctx, prompt) {
		switch ev.Type {
		case agent.EventToolCall:
			fmt.Printf("\n→ %s %s\n", ev.ToolName, ev.ToolInput)
		case agent.EventToolResult:
			status := "ok"
			if ev.ToolError {
				status = "error"
			}
			fmt.Printf("← %s (%s, %d bytes)\n", ev.ToolName, status, len(ev.Text))
		case agent.EventText:
			fmt.Println(ev.Text)
		case agent.EventRunEnd:
			fmt.Printf("\n[end_turn] in=%d out=%d cache_read=%d\n", ev.Usage.InputTokens, ev.Usage.OutputTokens, ev.Usage.CacheReadTokens)
		case agent.EventError:
			fmt.Fprintf(os.Stderr, "\n[error] %v (in=%d out=%d)\n", ev.Err, ev.Usage.InputTokens, ev.Usage.OutputTokens)
			code = 1
		}
	}
	return code
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
