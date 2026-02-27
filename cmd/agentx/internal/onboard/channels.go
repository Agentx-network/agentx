package onboard

type tokenField struct {
	Label       string
	Placeholder string
	ConfigField string
}

type channelInfo struct {
	Name        string
	ID          string
	TokenFields []tokenField
	HelpURL     string
}

var channels = []channelInfo{
	{
		Name: "Telegram",
		ID:   "telegram",
		TokenFields: []tokenField{
			{Label: "Bot Token", Placeholder: "123456:ABC-DEF...", ConfigField: "token"},
		},
		HelpURL: "https://core.telegram.org/bots#botfather",
	},
	{
		Name: "Discord",
		ID:   "discord",
		TokenFields: []tokenField{
			{Label: "Bot Token", Placeholder: "your-discord-bot-token", ConfigField: "token"},
		},
		HelpURL: "https://discord.com/developers/applications",
	},
	{
		Name: "Slack",
		ID:   "slack",
		TokenFields: []tokenField{
			{Label: "Bot Token (xoxb-...)", Placeholder: "xoxb-...", ConfigField: "bot_token"},
			{Label: "App Token (xapp-...)", Placeholder: "xapp-...", ConfigField: "app_token"},
		},
		HelpURL: "https://api.slack.com/apps",
	},
	{
		Name: "WhatsApp",
		ID:   "whatsapp",
		TokenFields: []tokenField{
			{Label: "Bridge URL", Placeholder: "ws://localhost:3001", ConfigField: "bridge_url"},
		},
		HelpURL: "https://github.com/nicechatx/chat-to-api",
	},
}

func findChannel(id string) *channelInfo {
	for i := range channels {
		if channels[i].ID == id {
			return &channels[i]
		}
	}
	return nil
}
