package onboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOnboardCommand(t *testing.T) {
	cmd := NewOnboardCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "onboard", cmd.Use)
	assert.Equal(t, "Initialize agentx configuration and workspace", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)

	assert.Len(t, cmd.Aliases, 1)
	assert.True(t, cmd.HasAlias("o"))

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.Nil(t, cmd.PersistentPreRun)
	assert.Nil(t, cmd.PersistentPostRun)

	assert.True(t, cmd.HasFlags())
	assert.False(t, cmd.HasSubCommands())

	// Check flags exist
	providerFlag := cmd.Flags().Lookup("provider")
	require.NotNil(t, providerFlag)
	assert.Equal(t, "", providerFlag.DefValue)

	apiKeyFlag := cmd.Flags().Lookup("api-key")
	require.NotNil(t, apiKeyFlag)
	assert.Equal(t, "", apiKeyFlag.DefValue)
}
