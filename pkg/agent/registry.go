package agent

import (
	"sync"

	"github.com/Agentx-network/agentx/pkg/config"
	"github.com/Agentx-network/agentx/pkg/logger"
	"github.com/Agentx-network/agentx/pkg/providers"
	"github.com/Agentx-network/agentx/pkg/routing"
)

// initFantasyModels initializes Fantasy models for all agents in the registry.
func (r *AgentRegistry) initFantasyModels(cfg *config.Config) {
	for id, agent := range r.agents {
		model, err := providers.FantasyModelFromFullConfig(cfg)
		if err != nil {
			logger.WarnCF("agent", "Failed to create Fantasy model, using legacy provider",
				map[string]any{
					"agent_id": id,
					"error":    err.Error(),
				})
			continue
		}

		// Wrap with fallback if configured
		if len(agent.Fallbacks) > 0 {
			cooldown := providers.NewCooldownTracker()
			var fallbackCfgs []*config.ModelConfig
			for _, fb := range agent.Fallbacks {
				fbCfg, err := cfg.GetModelConfig(fb)
				if err != nil {
					// Try treating as protocol/model directly
					fbCfg = &config.ModelConfig{
						Model:     fb,
						ModelName: fb,
					}
				}
				fallbackCfgs = append(fallbackCfgs, fbCfg)
			}
			if len(fallbackCfgs) > 0 {
				model = providers.NewFallbackLanguageModel(model, fallbackCfgs, cooldown)
			}
		}

		agent.FantasyModel = model
		logger.InfoCF("agent", "Fantasy model initialized",
			map[string]any{
				"agent_id": id,
				"provider": model.Provider(),
				"model":    model.Model(),
			})
	}
}

// AgentRegistry manages multiple agent instances and routes messages to them.
type AgentRegistry struct {
	agents   map[string]*AgentInstance
	resolver *routing.RouteResolver
	mu       sync.RWMutex
}

// NewAgentRegistry creates a registry from config, instantiating all agents.
func NewAgentRegistry(
	cfg *config.Config,
	provider providers.LLMProvider,
) *AgentRegistry {
	registry := &AgentRegistry{
		agents:   make(map[string]*AgentInstance),
		resolver: routing.NewRouteResolver(cfg),
	}

	agentConfigs := cfg.Agents.List
	if len(agentConfigs) == 0 {
		implicitAgent := &config.AgentConfig{
			ID:      "main",
			Default: true,
		}
		instance := NewAgentInstance(implicitAgent, &cfg.Agents.Defaults, cfg, provider)
		registry.agents["main"] = instance
		logger.InfoCF("agent", "Created implicit main agent (no agents.list configured)", nil)
	} else {
		for i := range agentConfigs {
			ac := &agentConfigs[i]
			id := routing.NormalizeAgentID(ac.ID)
			instance := NewAgentInstance(ac, &cfg.Agents.Defaults, cfg, provider)
			registry.agents[id] = instance
			logger.InfoCF("agent", "Registered agent",
				map[string]any{
					"agent_id":  id,
					"name":      ac.Name,
					"workspace": instance.Workspace,
					"model":     instance.Model,
				})
		}
	}

	// Initialize Fantasy models for all agents
	registry.initFantasyModels(cfg)

	return registry
}

// GetAgent returns the agent instance for a given ID.
func (r *AgentRegistry) GetAgent(agentID string) (*AgentInstance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id := routing.NormalizeAgentID(agentID)
	agent, ok := r.agents[id]
	return agent, ok
}

// ResolveRoute determines which agent handles the message.
func (r *AgentRegistry) ResolveRoute(input routing.RouteInput) routing.ResolvedRoute {
	return r.resolver.ResolveRoute(input)
}

// ListAgentIDs returns all registered agent IDs.
func (r *AgentRegistry) ListAgentIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.agents))
	for id := range r.agents {
		ids = append(ids, id)
	}
	return ids
}

// CanSpawnSubagent checks if parentAgentID is allowed to spawn targetAgentID.
func (r *AgentRegistry) CanSpawnSubagent(parentAgentID, targetAgentID string) bool {
	parent, ok := r.GetAgent(parentAgentID)
	if !ok {
		return false
	}
	if parent.Subagents == nil || parent.Subagents.AllowAgents == nil {
		return false
	}
	targetNorm := routing.NormalizeAgentID(targetAgentID)
	for _, allowed := range parent.Subagents.AllowAgents {
		if allowed == "*" {
			return true
		}
		if routing.NormalizeAgentID(allowed) == targetNorm {
			return true
		}
	}
	return false
}

// GetDefaultAgent returns the default agent instance.
func (r *AgentRegistry) GetDefaultAgent() *AgentInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if agent, ok := r.agents["main"]; ok {
		return agent
	}
	for _, agent := range r.agents {
		return agent
	}
	return nil
}
