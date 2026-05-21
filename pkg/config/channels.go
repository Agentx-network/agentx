package config

import "strings"

// PushChannels lists the channels the gateway can proactively deliver to (they
// have a registered Send path in the channel manager). The desktop/CLI are
// request/response only — they are deliberately excluded, because a delayed
// reminder or notification can never reach them.
var PushChannels = []string{
	"telegram", "discord", "slack", "whatsapp", "feishu",
	"dingtalk", "qq", "line", "onebot", "wecom", "wecom_app", "maixcam",
}

// ChannelEnabled reports whether the named channel is enabled in config.
func (c *Config) ChannelEnabled(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "telegram":
		return c.Channels.Telegram.Enabled
	case "discord":
		return c.Channels.Discord.Enabled
	case "slack":
		return c.Channels.Slack.Enabled
	case "whatsapp":
		return c.Channels.WhatsApp.Enabled
	case "feishu":
		return c.Channels.Feishu.Enabled
	case "dingtalk":
		return c.Channels.DingTalk.Enabled
	case "qq":
		return c.Channels.QQ.Enabled
	case "line":
		return c.Channels.LINE.Enabled
	case "onebot":
		return c.Channels.OneBot.Enabled
	case "wecom":
		return c.Channels.WeCom.Enabled
	case "wecom_app":
		return c.Channels.WeComApp.Enabled
	case "maixcam":
		return c.Channels.MaixCam.Enabled
	default:
		return false
	}
}

// ChannelAllowFrom returns the configured allow-list (permitted sender IDs) for
// the named channel. The first entry is treated as the owner's chat ID when we
// need a default delivery target.
func (c *Config) ChannelAllowFrom(name string) []string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "telegram":
		return c.Channels.Telegram.AllowFrom
	case "discord":
		return c.Channels.Discord.AllowFrom
	case "slack":
		return c.Channels.Slack.AllowFrom
	case "whatsapp":
		return c.Channels.WhatsApp.AllowFrom
	case "feishu":
		return c.Channels.Feishu.AllowFrom
	case "dingtalk":
		return c.Channels.DingTalk.AllowFrom
	case "qq":
		return c.Channels.QQ.AllowFrom
	case "line":
		return c.Channels.LINE.AllowFrom
	case "onebot":
		return c.Channels.OneBot.AllowFrom
	case "wecom":
		return c.Channels.WeCom.AllowFrom
	case "wecom_app":
		return c.Channels.WeComApp.AllowFrom
	case "maixcam":
		return c.Channels.MaixCam.AllowFrom
	default:
		return nil
	}
}

// OwnerChatID returns a sensible default delivery target ("owner") for a push
// channel: the first entry of its allow-list. Empty if none configured.
func (c *Config) OwnerChatID(name string) string {
	for _, id := range c.ChannelAllowFrom(name) {
		id = strings.TrimSpace(id)
		if id != "" && id != "*" {
			// allow_from entries can be "id|username"; the id part is the target.
			if i := strings.Index(id, "|"); i > 0 {
				return id[:i]
			}
			return id
		}
	}
	return ""
}

// SetChannelAllowFrom replaces the allow-list for the named channel. Returns
// false if the channel name is unknown.
func (c *Config) SetChannelAllowFrom(name string, list []string) bool {
	fsl := FlexibleStringSlice(list)
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "telegram":
		c.Channels.Telegram.AllowFrom = fsl
	case "discord":
		c.Channels.Discord.AllowFrom = fsl
	case "slack":
		c.Channels.Slack.AllowFrom = fsl
	case "whatsapp":
		c.Channels.WhatsApp.AllowFrom = fsl
	case "feishu":
		c.Channels.Feishu.AllowFrom = fsl
	case "dingtalk":
		c.Channels.DingTalk.AllowFrom = fsl
	case "qq":
		c.Channels.QQ.AllowFrom = fsl
	case "line":
		c.Channels.LINE.AllowFrom = fsl
	case "onebot":
		c.Channels.OneBot.AllowFrom = fsl
	case "wecom":
		c.Channels.WeCom.AllowFrom = fsl
	case "wecom_app":
		c.Channels.WeComApp.AllowFrom = fsl
	case "maixcam":
		c.Channels.MaixCam.AllowFrom = fsl
	default:
		return false
	}
	return true
}

// IsPushChannel reports whether the named channel can receive proactive
// (delayed/scheduled) messages.
func (c *Config) IsPushChannel(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	for _, ch := range PushChannels {
		if ch == n {
			return true
		}
	}
	return false
}

// FirstConnectedPushChannel returns the first enabled push channel that also
// has an owner chat ID, or ("","") if none — i.e. the best target for a
// proactive reminder when the user is on a non-deliverable surface (desktop/CLI).
func (c *Config) FirstConnectedPushChannel() (channel, ownerChatID string) {
	for _, ch := range PushChannels {
		if c.ChannelEnabled(ch) {
			if owner := c.OwnerChatID(ch); owner != "" {
				return ch, owner
			}
		}
	}
	return "", ""
}
