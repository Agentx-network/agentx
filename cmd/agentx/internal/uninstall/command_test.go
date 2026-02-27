package uninstall

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUninstallCommand(t *testing.T) {
	cmd := NewUninstallCommand()

	require.NotNil(t, cmd)

	assert.Equal(t, "uninstall", cmd.Use)
	assert.Equal(t, "Remove agentx and all its data from this system", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)

	assert.Len(t, cmd.Aliases, 1)
	assert.True(t, cmd.HasAlias("remove"))

	assert.Nil(t, cmd.Run)
	assert.NotNil(t, cmd.RunE)

	assert.True(t, cmd.HasFlags())
	assert.False(t, cmd.HasSubCommands())

	yesFlag := cmd.Flags().Lookup("yes")
	require.NotNil(t, yesFlag)
	assert.Equal(t, "false", yesFlag.DefValue)
	assert.Equal(t, "y", yesFlag.Shorthand)
}

func TestRemoveDataDir(t *testing.T) {
	// Create a temp directory to act as ~/.agentx
	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, ".agentx")
	require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "workspace"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "config.yaml"), []byte("key: val"), 0o644))

	// Override home via env so removeDataDir finds our temp dir.
	t.Setenv("HOME", tmp)

	err := removeDataDir()
	require.NoError(t, err)

	_, statErr := os.Stat(dataDir)
	assert.True(t, os.IsNotExist(statErr))
}

func TestRemoveDataDir_NotExist(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	err := removeDataDir()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestRemoveGatewayService_NoSystemd(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("skipping systemd test on macOS")
	}
	// Point PATH to an empty dir so systemctl is not found.
	tmp := t.TempDir()
	t.Setenv("PATH", tmp)

	err := removeSystemdService()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "systemctl not found")
}

func TestRemoveLaunchdService_NoPlist(t *testing.T) {
	// Use a temp dir as HOME so the plist path doesn't exist.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Should succeed gracefully when plist doesn't exist.
	err := removeLaunchdService()
	assert.NoError(t, err)
}

func TestRemoveGatewayService_Dispatches(t *testing.T) {
	// Verify removeGatewayService dispatches based on runtime.GOOS.
	// We can't change GOOS at runtime, but we verify it doesn't panic.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if runtime.GOOS != "darwin" {
		t.Setenv("PATH", tmp) // hide systemctl
	}

	err := removeGatewayService()
	if runtime.GOOS == "darwin" {
		// On macOS: removeLaunchdService with no plist → no error
		assert.NoError(t, err)
	} else {
		// On Linux without systemctl → error
		assert.Error(t, err)
	}
}
