package agent

import (
	"fmt"
	"time"

	"orbit/internal/core"
)

// Budget caps a single run. Zero values mean "unlimited".
type Budget struct {
	MaxTurns  int
	MaxTokens int
	MaxTime   time.Duration
}

// Check reports why the run must stop, or nil if it may continue.
func (b Budget) Check(turns int, usage core.Usage, elapsed time.Duration) error {
	if b.MaxTurns > 0 && turns >= b.MaxTurns {
		return fmt.Errorf("budget exceeded: %d turns", turns)
	}
	if total := usage.InputTokens + usage.OutputTokens; b.MaxTokens > 0 && total >= b.MaxTokens {
		return fmt.Errorf("budget exceeded: %d tokens", total)
	}
	if b.MaxTime > 0 && elapsed >= b.MaxTime {
		return fmt.Errorf("budget exceeded: %s elapsed", elapsed.Round(time.Second))
	}
	return nil
}
