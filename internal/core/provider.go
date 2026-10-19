package core

import "context"

// Provider is the stateless contract every LLM backend implements.
// It knows nothing about turns, budgets or history beyond the request it is given.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
