// Package constants provides shared constants across the codebase.
package constants

// Internal channel identifiers. "cli" reaches the terminal directly,
// "system" is for routing system messages, "subagent" for spawned agents.
const (
	ChannelCLI      = "cli"
	ChannelSystem   = "system"
	ChannelSubagent = "subagent"
)

// internalChannels defines channels that are used for internal communication
// and should not be exposed to external users or recorded as last active channel.
var internalChannels = map[string]struct{}{
	ChannelCLI:      {},
	ChannelSystem:   {},
	ChannelSubagent: {},
}

// IsInternalChannel returns true if the channel is an internal channel.
func IsInternalChannel(channel string) bool {
	_, found := internalChannels[channel]
	return found
}
