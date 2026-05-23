package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Agentx-network/agentx/pkg/bus"
	"github.com/Agentx-network/agentx/pkg/config"
	"github.com/Agentx-network/agentx/pkg/cron"
	"github.com/Agentx-network/agentx/pkg/utils"
)

// JobExecutor is the interface for executing cron jobs through the agent
type JobExecutor interface {
	ProcessDirectWithChannel(ctx context.Context, content, sessionKey, channel, chatID string) (string, error)
}

// CronTool provides scheduling capabilities for the agent
type CronTool struct {
	cronService      *cron.CronService
	executor         JobExecutor
	msgBus           *bus.MessageBus
	execTool         *ExecTool
	cfg              *config.Config
	cfgPath          string // when set, reminder targeting reads live config from here
	channel          string
	chatID           string
	scheduledInRound bool // whether a job was added during the current processing round
	mu               sync.RWMutex
}

// liveConfig returns the freshest config: re-read from disk when cfgPath is set
// (so an owner claimed mid-session is visible), else the startup config.
func (t *CronTool) liveConfig() *config.Config {
	if t.cfgPath != "" {
		if live, err := config.LoadConfig(t.cfgPath); err == nil {
			return live
		}
	}
	return t.cfg
}

// NewCronTool creates a new CronTool
// execTimeout: 0 means no timeout, >0 sets the timeout duration
func NewCronTool(
	cronService *cron.CronService, executor JobExecutor, msgBus *bus.MessageBus, workspace string, restrict bool,
	execTimeout time.Duration, cfg *config.Config,
) *CronTool {
	execTool := NewExecToolWithConfig(workspace, restrict, cfg)
	execTool.SetTimeout(execTimeout)
	return &CronTool{
		cronService: cronService,
		executor:    executor,
		msgBus:      msgBus,
		execTool:    execTool,
		cfg:         cfg,
		cfgPath:     config.DefaultConfigPath(),
	}
}

// resolveReminderTarget decides where a scheduled reminder should actually be
// delivered. Delayed messages can only reach a "push" channel (Telegram, etc.)
// — the desktop/CLI are request/response only, so a job left targeting them is
// silently dropped by the channel manager. So:
//   - if the user is already on an enabled push channel, keep it;
//   - otherwise redirect to the first connected push channel + its owner chat ID;
//   - if nothing is connected, return an honest error instead of scheduling a
//     reminder that can never be delivered.
//
// resolveReminderTarget decides where a delayed reminder will be delivered.
//   - requested != "": the user explicitly named a channel (e.g. "ping me on
//     Telegram") → it must be a connected push channel with a known owner ID.
//   - otherwise: deliver wherever the user is. The desktop chat is a valid
//     target (the gateway queues it and the app polls), so a plain "remind me…"
//     from the desktop fires back into that same chat — no Telegram needed.
func (t *CronTool) resolveReminderTarget(sessChannel, sessChatID, requested string) (channel, chatID string, errResult *ToolResult) {
	cfg := t.liveConfig()
	requested = strings.ToLower(strings.TrimSpace(requested))

	// Explicit channel requested by the user.
	if requested != "" && requested != sessChannel {
		if cfg != nil && cfg.IsPushChannel(requested) {
			if !cfg.ChannelEnabled(requested) {
				return "", "", BlockedResult(
					fmt.Sprintf("The %s channel isn't connected yet — set it up in Config → Channels, then I can ping you there.", requested),
					fmt.Sprintf("Requested channel %q is not enabled. Tell the user to connect it in Config → Channels; do NOT claim the reminder was scheduled.", requested),
				)
			}
			owner := cfg.OwnerChatID(requested)
			if owner == "" {
				return "", "", BlockedResult(
					fmt.Sprintf("I don't know your %s chat ID yet — message the bot on %s once so it learns your ID, then I can reach you there.", requested, requested),
					fmt.Sprintf("Requested channel %q has no owner chat ID. Tell the user to message the bot once; do NOT claim the reminder was scheduled.", requested),
				)
			}
			return requested, owner, nil
		}
		// Unknown/non-push requested channel → fall through to defaults below.
	}

	// Current session is itself a connected push channel → use it directly.
	if cfg != nil && cfg.IsPushChannel(sessChannel) && cfg.ChannelEnabled(sessChannel) && sessChatID != "" {
		return sessChannel, sessChatID, nil
	}

	// Desktop is a valid local delivery target (gateway queues; the app polls).
	if sessChannel == "desktop" {
		cid := sessChatID
		if cid == "" {
			cid = "chat"
		}
		return "desktop", cid, nil
	}

	// Other surfaces (e.g. CLI) can't receive a delayed push of their own — try
	// any connected push channel before giving up.
	if cfg != nil {
		if ch, owner := cfg.FirstConnectedPushChannel(); ch != "" {
			return ch, owner, nil
		}
	}
	return "", "", BlockedResult(
		"I can only send a reminder to a connected channel like Telegram, and none is set up yet. "+
			"Connect one in Config → Channels (and put your chat ID in 'Allowed senders'), then I can ping you.",
		"No deliverable channel is available for this surface. Tell the user to connect a push channel; "+
			"do NOT claim the reminder was scheduled.",
	)
}

// Name returns the tool name
func (t *CronTool) Name() string {
	return "cron"
}

// Description returns the tool description
func (t *CronTool) Description() string {
	return "Schedule reminders, pings, alerts, tasks, or system commands for a future time or interval. " +
		"IMPORTANT: for ANY 'remind me', 'ping me', 'alert me', 'notify me' at/in/every <time> request you MUST use this tool " +
		"(NEVER spawn — a subagent can't wait). Use 'at_seconds' for one-time (e.g. 'ping me in 2 minutes' → at_seconds=120). " +
		"Use 'every_seconds' for recurring (e.g. 'every 2 hours' → every_seconds=7200). Use 'cron_expr' for complex schedules. " +
		"Use 'command' to run a shell command on schedule. Delivery goes to a connected push channel (e.g. Telegram) automatically."
}

// Parameters returns the tool parameters schema
func (t *CronTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"add", "list", "remove", "enable", "disable"},
				"description": "Action to perform. Use 'add' when user wants to schedule a reminder or task.",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "The reminder/task message to display when triggered. If 'command' is used, this describes what the command does.",
			},
			"command": map[string]any{
				"type":        "string",
				"description": "Optional: Shell command to execute directly (e.g., 'df -h'). If set, the agent will run this command and report output instead of just showing the message. 'deliver' will be forced to false for commands.",
			},
			"at_seconds": map[string]any{
				"type":        "integer",
				"description": "One-time reminder: seconds from now when to trigger (e.g., 600 for 10 minutes later). Use this for one-time reminders like 'remind me in 10 minutes'.",
			},
			"every_seconds": map[string]any{
				"type":        "integer",
				"description": "Recurring interval in seconds (e.g., 3600 for every hour). Use this ONLY for recurring tasks like 'every 2 hours' or 'daily reminder'.",
			},
			"cron_expr": map[string]any{
				"type":        "string",
				"description": "Cron expression for complex recurring schedules (e.g., '0 9 * * *' for daily at 9am). Use this for complex recurring schedules.",
			},
			"job_id": map[string]any{
				"type":        "string",
				"description": "Job ID (for remove/enable/disable)",
			},
			"channel": map[string]any{
				"type":        "string",
				"description": "Optional: where to deliver the reminder. Set ONLY if the user explicitly named a channel (e.g. 'telegram' for \"ping me on Telegram\"). Leave EMPTY to deliver wherever the user is right now (e.g. the desktop chat).",
			},
			"deliver": map[string]any{
				"type":        "boolean",
				"description": "If true, send message directly to channel. If false, let agent process message (for complex tasks). Default: true",
			},
		},
		"required": []string{"action"},
	}
}

// SetContext sets the current session context for job creation. It also resets
// the per-round "scheduled" flag (called at the start of each processing round).
func (t *CronTool) SetContext(channel, chatID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.channel = channel
	t.chatID = chatID
	t.scheduledInRound = false
}

// HasScheduledInRound reports whether a job was added during the current round.
// Used by the deterministic reminder fallback to avoid double-scheduling when
// the model already created the cron job itself.
func (t *CronTool) HasScheduledInRound() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.scheduledInRound
}

// Execute runs the tool with the given arguments
func (t *CronTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	switch action {
	case "add":
		return t.addJob(args)
	case "list":
		return t.listJobs()
	case "remove":
		return t.removeJob(args)
	case "enable":
		return t.enableJob(args, true)
	case "disable":
		return t.enableJob(args, false)
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

func (t *CronTool) addJob(args map[string]any) *ToolResult {
	t.mu.RLock()
	sessChannel := t.channel
	sessChatID := t.chatID
	t.mu.RUnlock()

	message, ok := args["message"].(string)
	if !ok || message == "" {
		return ErrorResult("message is required for add")
	}

	// Resolve where this reminder can actually be delivered. A command-only job
	// (no chat delivery) keeps the session context; a chat reminder must target
	// a push channel that can receive it later.
	command, _ := args["command"].(string)
	requestedChannel, _ := args["channel"].(string)
	channel, chatID := sessChannel, sessChatID
	if command == "" {
		var errResult *ToolResult
		channel, chatID, errResult = t.resolveReminderTarget(sessChannel, sessChatID, requestedChannel)
		if errResult != nil {
			return errResult
		}
	}

	var schedule cron.CronSchedule

	// Check for at_seconds (one-time), every_seconds (recurring), or cron_expr
	atSeconds, hasAt := args["at_seconds"].(float64)
	everySeconds, hasEvery := args["every_seconds"].(float64)
	cronExpr, hasCron := args["cron_expr"].(string)

	// Priority: at_seconds > every_seconds > cron_expr
	if hasAt {
		atMS := time.Now().UnixMilli() + int64(atSeconds)*1000
		schedule = cron.CronSchedule{
			Kind: "at",
			AtMS: &atMS,
		}
	} else if hasEvery {
		everyMS := int64(everySeconds) * 1000
		schedule = cron.CronSchedule{
			Kind:    "every",
			EveryMS: &everyMS,
		}
	} else if hasCron {
		schedule = cron.CronSchedule{
			Kind: "cron",
			Expr: cronExpr,
		}
	} else {
		return ErrorResult("one of at_seconds, every_seconds, or cron_expr is required")
	}

	// Read deliver parameter, default to true
	deliver := true
	if d, ok := args["deliver"].(bool); ok {
		deliver = d
	}

	if command != "" {
		// Commands run through the exec tool, not delivered as a chat message.
		deliver = false
	}

	// Truncate message for job name (max 30 chars)
	messagePreview := utils.Truncate(message, 30)

	job, err := t.cronService.AddJob(
		messagePreview,
		schedule,
		message,
		deliver,
		channel,
		chatID,
	)
	if err != nil {
		return BlockedResult(
			"I couldn't schedule that — the timing wasn't valid. Try a plain interval (e.g. \"every 30 minutes\") or a clear time, and I'll set it up.",
			fmt.Sprintf("cron AddJob failed: %v. Re-derive a valid schedule (at_seconds/every_seconds/cron_expr) and retry.", err),
		)
	}

	if command != "" {
		job.Payload.Command = command
		// Need to save the updated payload
		t.cronService.UpdateJob(job)
	}

	// Mark that a job was scheduled this round (the deterministic reminder
	// fallback checks this to avoid double-scheduling).
	t.mu.Lock()
	t.scheduledInRound = true
	t.mu.Unlock()

	// Tell the model the reminder is scheduled — kept SHORT on purpose so the
	// model produces a brief human confirmation, not a verbose dump with job
	// IDs, delivery metadata, or unsolicited follow-up questions.
	if command == "" && channel == "desktop" {
		return SilentResult("Reminder scheduled. Reply with one short sentence confirming it; do NOT mention the job ID, internal IDs, or ask to add another.")
	}
	if command == "" && channel != sessChannel {
		return SilentResult(fmt.Sprintf(
			"Reminder scheduled on %s. Reply with one short sentence confirming it; do NOT mention the job ID, internal IDs, or ask to add another.",
			channel,
		))
	}
	return SilentResult(fmt.Sprintf("Scheduled job '%s' on %s. Reply briefly; no job ID, no follow-up offer.", job.Name, channel))
}

func (t *CronTool) listJobs() *ToolResult {
	jobs := t.cronService.ListJobs(false)

	if len(jobs) == 0 {
		return SilentResult("No scheduled jobs")
	}

	result := "Scheduled jobs:\n"
	for _, j := range jobs {
		var scheduleInfo string
		if j.Schedule.Kind == "every" && j.Schedule.EveryMS != nil {
			scheduleInfo = fmt.Sprintf("every %ds", *j.Schedule.EveryMS/1000)
		} else if j.Schedule.Kind == "cron" {
			scheduleInfo = j.Schedule.Expr
		} else if j.Schedule.Kind == "at" {
			scheduleInfo = "one-time"
		} else {
			scheduleInfo = "unknown"
		}
		result += fmt.Sprintf("- %s (id: %s, %s)\n", j.Name, j.ID, scheduleInfo)
	}

	return SilentResult(result)
}

func (t *CronTool) removeJob(args map[string]any) *ToolResult {
	jobID, ok := args["job_id"].(string)
	if !ok || jobID == "" {
		return ErrorResult("job_id is required for remove")
	}

	if t.cronService.RemoveJob(jobID) {
		return SilentResult(fmt.Sprintf("Cron job removed: %s", jobID))
	}
	return ErrorResult(fmt.Sprintf("Job %s not found", jobID))
}

func (t *CronTool) enableJob(args map[string]any, enable bool) *ToolResult {
	jobID, ok := args["job_id"].(string)
	if !ok || jobID == "" {
		return ErrorResult("job_id is required for enable/disable")
	}

	job := t.cronService.EnableJob(jobID, enable)
	if job == nil {
		return ErrorResult(fmt.Sprintf("Job %s not found", jobID))
	}

	status := "enabled"
	if !enable {
		status = "disabled"
	}
	return SilentResult(fmt.Sprintf("Cron job '%s' %s", job.Name, status))
}

// ExecuteJob executes a cron job through the agent
func (t *CronTool) ExecuteJob(ctx context.Context, job *cron.CronJob) string {
	// Get channel/chatID from job payload
	channel := job.Payload.Channel
	chatID := job.Payload.To

	// Default values if not set
	if channel == "" {
		channel = "cli"
	}
	if chatID == "" {
		chatID = "direct"
	}

	// Execute command if present
	if job.Payload.Command != "" {
		args := map[string]any{
			"command": job.Payload.Command,
		}

		result := t.execTool.Execute(ctx, args)
		var output string
		if result.IsError {
			output = fmt.Sprintf("Error executing scheduled command: %s", result.ForLLM)
		} else {
			output = fmt.Sprintf("Scheduled command '%s' executed:\n%s", job.Payload.Command, result.ForLLM)
		}

		t.msgBus.PublishOutbound(bus.OutboundMessage{
			Channel: channel,
			ChatID:  chatID,
			Content: output,
		})
		return "ok"
	}

	// If deliver=true, send message directly without agent processing
	if job.Payload.Deliver {
		t.msgBus.PublishOutbound(bus.OutboundMessage{
			Channel: channel,
			ChatID:  chatID,
			Content: job.Payload.Message,
		})
		return "ok"
	}

	// For deliver=false, process through agent (for complex tasks)
	sessionKey := fmt.Sprintf("cron-%s", job.ID)

	// Call agent with job's message
	response, err := t.executor.ProcessDirectWithChannel(
		ctx,
		job.Payload.Message,
		sessionKey,
		channel,
		chatID,
	)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	// Response is automatically sent via MessageBus by AgentLoop
	_ = response // Will be sent by AgentLoop
	return "ok"
}
