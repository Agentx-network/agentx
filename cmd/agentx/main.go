// AgentX - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 AgentX contributors

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Agentx-network/agentx/cmd/agentx/internal"
	"github.com/Agentx-network/agentx/cmd/agentx/internal/agent"
	"github.com/Agentx-network/agentx/cmd/agentx/internal/auth"
	"github.com/Agentx-network/agentx/cmd/agentx/internal/cron"
	"github.com/Agentx-network/agentx/cmd/agentx/internal/gateway"
	"github.com/Agentx-network/agentx/cmd/agentx/internal/migrate"
	"github.com/Agentx-network/agentx/cmd/agentx/internal/onboard"
	"github.com/Agentx-network/agentx/cmd/agentx/internal/uninstall"
	"github.com/Agentx-network/agentx/cmd/agentx/internal/skills"
	"github.com/Agentx-network/agentx/cmd/agentx/internal/status"
	"github.com/Agentx-network/agentx/cmd/agentx/internal/version"
)

func NewAgentxCommand() *cobra.Command {
	short := fmt.Sprintf("%s agentx - Personal AI Assistant v%s\n\n", internal.Logo, internal.GetVersion())

	cmd := &cobra.Command{
		Use:     "agentx",
		Short:   short,
		Example: "agentx list",
	}

	cmd.AddCommand(
		onboard.NewOnboardCommand(),
		agent.NewAgentCommand(),
		auth.NewAuthCommand(),
		gateway.NewGatewayCommand(),
		status.NewStatusCommand(),
		cron.NewCronCommand(),
		migrate.NewMigrateCommand(),
		skills.NewSkillsCommand(),
		version.NewVersionCommand(),
		uninstall.NewUninstallCommand(),
	)

	return cmd
}

func main() {
	cmd := NewAgentxCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
