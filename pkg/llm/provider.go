package llm

import "context"

type GenerateRequest struct {
	Prompt string
}

type GenerateResponse struct {
	Content string
}

type Provider interface {
	Generate(ctx context.Context, request GenerateRequest) (GenerateResponse, error)
}
