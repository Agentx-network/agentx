package providers

import (
	"context"
	"fmt"
	"strings"

	"charm.land/fantasy"

	"github.com/Agentx-network/agentx/pkg/config"
	"github.com/Agentx-network/agentx/pkg/logger"
)

// FallbackLanguageModel wraps multiple Fantasy LanguageModels and tries them in order.
// It delegates to the existing FallbackChain for cooldown and classification logic.
type FallbackLanguageModel struct {
	primary    fantasy.LanguageModel
	candidates []fallbackModelCandidate
	cooldown   *CooldownTracker
}

type fallbackModelCandidate struct {
	model    fantasy.LanguageModel
	provider string
	modelID  string
}

// NewFallbackLanguageModel creates a FallbackLanguageModel from a primary model and fallback configs.
func NewFallbackLanguageModel(
	primary fantasy.LanguageModel,
	fallbackCfgs []*config.ModelConfig,
	cooldown *CooldownTracker,
) *FallbackLanguageModel {
	candidates := make([]fallbackModelCandidate, 0, len(fallbackCfgs)+1)

	// Primary is always first
	candidates = append(candidates, fallbackModelCandidate{
		model:    primary,
		provider: primary.Provider(),
		modelID:  primary.Model(),
	})

	// Add fallbacks
	for _, cfg := range fallbackCfgs {
		model, err := FantasyModelFromConfig(cfg)
		if err != nil {
			logger.WarnCF("providers", "Failed to create fallback model",
				map[string]any{
					"model": cfg.Model,
					"error": err.Error(),
				})
			continue
		}
		candidates = append(candidates, fallbackModelCandidate{
			model:    model,
			provider: model.Provider(),
			modelID:  model.Model(),
		})
	}

	return &FallbackLanguageModel{
		primary:    primary,
		candidates: candidates,
		cooldown:   cooldown,
	}
}

// Generate tries each candidate until one succeeds.
func (f *FallbackLanguageModel) Generate(ctx context.Context, call fantasy.Call) (*fantasy.Response, error) {
	var lastErr error

	for i, candidate := range f.candidates {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if f.cooldown != nil && !f.cooldown.IsAvailable(candidate.provider) {
			logger.DebugCF("providers", "Skipping provider in cooldown",
				map[string]any{"provider": candidate.provider})
			continue
		}

		resp, err := candidate.model.Generate(ctx, call)
		if err == nil {
			if f.cooldown != nil {
				f.cooldown.MarkSuccess(candidate.provider)
			}
			if i > 0 {
				logger.InfoCF("providers", "Fallback succeeded",
					map[string]any{
						"provider": candidate.provider,
						"model":    candidate.modelID,
						"attempt":  i + 1,
					})
			}
			return resp, nil
		}

		lastErr = err
		errMsg := strings.ToLower(err.Error())

		// Non-retriable: format errors
		if strings.Contains(errMsg, "invalid") && strings.Contains(errMsg, "request") {
			return nil, err
		}

		// Mark failure and try next
		if f.cooldown != nil {
			f.cooldown.MarkFailure(candidate.provider, classifyFantasyError(errMsg))
		}

		logger.WarnCF("providers", "Fallback attempt failed",
			map[string]any{
				"provider": candidate.provider,
				"model":    candidate.modelID,
				"attempt":  i + 1,
				"error":    err.Error(),
			})
	}

	return nil, fmt.Errorf("all fallback candidates exhausted: %w", lastErr)
}

// Stream tries each candidate until one succeeds.
func (f *FallbackLanguageModel) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	var lastErr error

	for i, candidate := range f.candidates {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if f.cooldown != nil && !f.cooldown.IsAvailable(candidate.provider) {
			continue
		}

		resp, err := candidate.model.Stream(ctx, call)
		if err == nil {
			if f.cooldown != nil {
				f.cooldown.MarkSuccess(candidate.provider)
			}
			if i > 0 {
				logger.InfoCF("providers", "Stream fallback succeeded",
					map[string]any{
						"provider": candidate.provider,
						"model":    candidate.modelID,
						"attempt":  i + 1,
					})
			}
			return resp, nil
		}

		lastErr = err
		if f.cooldown != nil {
			f.cooldown.MarkFailure(candidate.provider, classifyFantasyError(strings.ToLower(err.Error())))
		}
	}

	return nil, fmt.Errorf("all stream fallback candidates exhausted: %w", lastErr)
}

// GenerateObject delegates to the primary model (no fallback for object generation).
func (f *FallbackLanguageModel) GenerateObject(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	return f.primary.GenerateObject(ctx, call)
}

// StreamObject delegates to the primary model.
func (f *FallbackLanguageModel) StreamObject(ctx context.Context, call fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return f.primary.StreamObject(ctx, call)
}

// Provider returns the primary provider name.
func (f *FallbackLanguageModel) Provider() string {
	return f.primary.Provider()
}

// Model returns the primary model name.
func (f *FallbackLanguageModel) Model() string {
	return f.primary.Model()
}

func classifyFantasyError(errMsg string) FailoverReason {
	switch {
	case strings.Contains(errMsg, "rate") || strings.Contains(errMsg, "429"):
		return FailoverRateLimit
	case strings.Contains(errMsg, "auth") || strings.Contains(errMsg, "401") || strings.Contains(errMsg, "403"):
		return FailoverAuth
	case strings.Contains(errMsg, "timeout"):
		return FailoverTimeout
	case strings.Contains(errMsg, "overloaded") || strings.Contains(errMsg, "529"):
		return FailoverOverloaded
	default:
		return FailoverUnknown
	}
}
