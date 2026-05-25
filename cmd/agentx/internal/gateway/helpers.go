package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/Agentx-network/agentx/cmd/agentx/internal"
	"github.com/Agentx-network/agentx/pkg/agent"
	"github.com/Agentx-network/agentx/pkg/bus"
	"github.com/Agentx-network/agentx/pkg/channels"
	"github.com/Agentx-network/agentx/pkg/config"
	"github.com/Agentx-network/agentx/pkg/cron"
	"github.com/Agentx-network/agentx/pkg/devices"
	"github.com/Agentx-network/agentx/pkg/health"
	"github.com/Agentx-network/agentx/pkg/heartbeat"
	"github.com/Agentx-network/agentx/pkg/logger"
	"github.com/Agentx-network/agentx/pkg/state"
	"github.com/Agentx-network/agentx/pkg/tools"
	"github.com/Agentx-network/agentx/pkg/voice"
)

func gatewayCmd(debug bool) error {
	if debug {
		logger.SetLevel(logger.DEBUG)
		fmt.Println("🔍 Debug mode enabled")
	}

	// Enable file logging to ~/.agentx/gateway.log
	home, err := os.UserHomeDir()
	if err == nil {
		logDir := filepath.Join(home, ".agentx")
		os.MkdirAll(logDir, 0o755)
		logPath := filepath.Join(logDir, "gateway.log")
		if err := logger.EnableFileLogging(logPath); err != nil {
			fmt.Printf("Warning: could not enable file logging: %v\n", err)
		} else {
			defer logger.DisableFileLogging()
		}
	}

	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, nil)

	// Print agent startup info
	fmt.Println("\n📦 Agent Status:")
	startupInfo := agentLoop.GetStartupInfo()
	toolsInfo := startupInfo["tools"].(map[string]any)
	skillsInfo := startupInfo["skills"].(map[string]any)
	fmt.Printf("  • Tools: %d loaded\n", toolsInfo["count"])
	fmt.Printf("  • Skills: %d/%d available\n",
		skillsInfo["available"],
		skillsInfo["total"])

	// Log to file as well
	logger.InfoCF("agent", "Agent initialized",
		map[string]any{
			"tools_count":      toolsInfo["count"],
			"skills_total":     skillsInfo["total"],
			"skills_available": skillsInfo["available"],
		})

	// Setup cron tool and service
	execTimeout := time.Duration(cfg.Tools.Cron.ExecTimeoutMinutes) * time.Minute
	cronService := setupCronTool(
		agentLoop,
		msgBus,
		cfg.WorkspacePath(),
		cfg.Agents.Defaults.RestrictToWorkspace,
		execTimeout,
		cfg,
	)

	heartbeatService := heartbeat.NewHeartbeatService(
		cfg.WorkspacePath(),
		cfg.Heartbeat.Interval,
		cfg.Heartbeat.Enabled,
	)
	heartbeatService.SetBus(msgBus)
	heartbeatService.SetHandler(func(prompt, channel, chatID string) *tools.ToolResult {
		// Use cli:direct as fallback if no valid channel
		if channel == "" || chatID == "" {
			channel, chatID = "cli", "direct"
		}
		// Use ProcessHeartbeat - no session history, each heartbeat is independent
		var response string
		response, err = agentLoop.ProcessHeartbeat(context.Background(), prompt, channel, chatID)
		if err != nil {
			return tools.ErrorResult(fmt.Sprintf("Heartbeat error: %v", err))
		}
		if response == "HEARTBEAT_OK" {
			return tools.SilentResult("Heartbeat OK")
		}
		// For heartbeat, always return silent - the subagent result will be
		// sent to user via processSystemMessage when the async task completes
		return tools.SilentResult(response)
	})

	channelManager, err := channels.NewManager(cfg, msgBus)
	if err != nil {
		return fmt.Errorf("error creating channel manager: %w", err)
	}

	// Inject channel manager into agent loop for command handling
	agentLoop.SetChannelManager(channelManager)

	// The desktop chat has no push connection — proactive messages (cron
	// reminders, async results) targeting the "desktop" channel are queued here
	// and drained by the desktop app via /api/notifications.
	notifier := newDesktopNotifier()
	channelManager.SetLocalDeliveryHook(func(msg bus.OutboundMessage) bool {
		if msg.Channel == "desktop" {
			notifier.enqueue(msg.ChatID, msg.Content)
			return true
		}
		return false
	})

	var transcriber *voice.GroqTranscriber
	groqAPIKey := cfg.Providers.Groq.APIKey
	if groqAPIKey == "" {
		for _, mc := range cfg.ModelList {
			if strings.HasPrefix(mc.Model, "groq/") && mc.APIKey != "" {
				groqAPIKey = mc.APIKey
				break
			}
		}
	}
	if groqAPIKey != "" {
		transcriber = voice.NewGroqTranscriber(groqAPIKey)
		logger.InfoC("voice", "Groq voice transcription enabled")
	}

	if transcriber != nil {
		if telegramChannel, ok := channelManager.GetChannel("telegram"); ok {
			if tc, ok := telegramChannel.(*channels.TelegramChannel); ok {
				tc.SetTranscriber(transcriber)
				logger.InfoC("voice", "Groq transcription attached to Telegram channel")
			}
		}
		if discordChannel, ok := channelManager.GetChannel("discord"); ok {
			if dc, ok := discordChannel.(*channels.DiscordChannel); ok {
				dc.SetTranscriber(transcriber)
				logger.InfoC("voice", "Groq transcription attached to Discord channel")
			}
		}
		if slackChannel, ok := channelManager.GetChannel("slack"); ok {
			if sc, ok := slackChannel.(*channels.SlackChannel); ok {
				sc.SetTranscriber(transcriber)
				logger.InfoC("voice", "Groq transcription attached to Slack channel")
			}
		}
	}

	enabledChannels := channelManager.GetEnabledChannels()
	if len(enabledChannels) > 0 {
		fmt.Printf("✓ Channels enabled: %s\n", enabledChannels)
	} else {
		fmt.Println("⚠ Warning: No channels enabled")
	}

	fmt.Printf("✓ Gateway started on %s:%d\n", cfg.Gateway.Host, cfg.Gateway.Port)
	fmt.Println("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cronService.Start(); err != nil {
		fmt.Printf("Error starting cron service: %v\n", err)
	}
	fmt.Println("✓ Cron service started")

	if err := heartbeatService.Start(); err != nil {
		fmt.Printf("Error starting heartbeat service: %v\n", err)
	}
	fmt.Println("✓ Heartbeat service started")

	stateManager := state.NewManager(cfg.WorkspacePath())
	deviceService := devices.NewService(devices.Config{
		Enabled:    cfg.Devices.Enabled,
		MonitorUSB: cfg.Devices.MonitorUSB,
	}, stateManager)
	deviceService.SetBus(msgBus)
	if err := deviceService.Start(ctx); err != nil {
		fmt.Printf("Error starting device service: %v\n", err)
	} else if cfg.Devices.Enabled {
		fmt.Println("✓ Device event service started")
	}

	if err := channelManager.StartAll(ctx); err != nil {
		fmt.Printf("Error starting channels: %v\n", err)
	}

	healthServer := health.NewServer(cfg.Gateway.Host, cfg.Gateway.Port)

	// Register chat API endpoint (SSE streaming)
	healthServer.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Message    string `json:"message"`
			SessionKey string `json:"sessionKey"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if req.Message == "" {
			http.Error(w, `{"error":"message is required"}`, http.StatusBadRequest)
			return
		}
		if req.SessionKey == "" {
			req.SessionKey = "desktop:chat"
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Subscribe to stream deltas for this request
		sub := &bus.StreamSubscriber{
			Ch: make(chan bus.StreamDelta, 200),
			Filter: func(d bus.StreamDelta) bool {
				return d.Channel == "desktop" && d.ChatID == "chat"
			},
		}
		msgBus.AddStreamSubscriber(sub)

		// Forward stream deltas as SSE events in background
		streamDone := make(chan struct{})
		go func() {
			defer close(streamDone)
			for delta := range sub.Ch {
				data, _ := json.Marshal(map[string]any{
					"type":  "delta",
					"delta": delta.Delta,
					"done":  delta.Done,
				})
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}()

		// Process the message (blocks until complete)
		response, procErr := agentLoop.ProcessDirectWithChannel(
			r.Context(), req.Message, req.SessionKey, "desktop", "chat",
		)

		// Unsubscribe and wait for goroutine to drain
		msgBus.RemoveStreamSubscriber(sub)
		<-streamDone

		// Send final event with full response
		if procErr != nil {
			// Show the desktop a brief, clean message (the full error is logged
			// gateway-side); avoids dumping raw provider quota/URL detail.
			data, _ := json.Marshal(map[string]any{
				"type":  "error",
				"error": agent.HumanizeError(procErr),
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
		} else {
			data, _ := json.Marshal(map[string]any{
				"type":     "done",
				"response": response,
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		flusher.Flush()
	})

	// Register notifications endpoint: the desktop app polls this to drain any
	// proactively-delivered messages (cron reminders fired while the user wasn't
	// mid-request). Returns and clears the queue for the given chat.
	healthServer.HandleFunc("/api/notifications", func(w http.ResponseWriter, r *http.Request) {
		chatID := r.URL.Query().Get("chatID")
		if chatID == "" {
			chatID = "chat"
		}
		messages := notifier.drain(chatID)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_ = json.NewEncoder(w).Encode(map[string]any{"messages": messages})
	})

	// Schedulers API: lets the desktop dashboard list and cancel scheduled
	// cron jobs. This is the out-of-band escape hatch for runaway crons (the
	// client report where a stuck 10-second command flooded the chat) — the
	// user can clear everything from the dashboard without having to interact
	// with a chat that's being spammed.
	//
	//   GET    /api/schedulers           → { schedulers: [...] }
	//   DELETE /api/schedulers?id=<id>   → { removed: 0 or 1 }
	//   DELETE /api/schedulers/all       → { removed: <n> }
	healthServer.HandleFunc("/api/schedulers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		switch r.Method {
		case http.MethodGet:
			jobs := cronService.ListJobs(true) // include disabled
			out := make([]map[string]any, 0, len(jobs))
			for i := range jobs {
				j := &jobs[i]
				entry := map[string]any{
					"id":          j.ID,
					"name":        j.Name,
					"enabled":     j.Enabled,
					"message":     j.Payload.Message,
					"command":     j.Payload.Command,
					"channel":     j.Payload.Channel,
					"chatId":      j.Payload.To,
					"kind":        j.Schedule.Kind,
					"createdAtMs": j.CreatedAtMS,
				}
				if j.Schedule.AtMS != nil {
					entry["atMs"] = *j.Schedule.AtMS
				}
				if j.Schedule.EveryMS != nil {
					entry["everyMs"] = *j.Schedule.EveryMS
				}
				if j.Schedule.Expr != "" {
					entry["cronExpr"] = j.Schedule.Expr
				}
				out = append(out, entry)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"schedulers": out})
		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			if id == "" {
				http.Error(w, `{"error":"id is required (or use /api/schedulers/all)"}`, http.StatusBadRequest)
				return
			}
			removed := 0
			if cronService.RemoveJob(id) {
				removed = 1
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"removed": removed})
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	healthServer.HandleFunc("/api/schedulers/all", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method != http.MethodDelete {
			http.Error(w, `{"error":"method not allowed; use DELETE"}`, http.StatusMethodNotAllowed)
			return
		}
		jobs := cronService.ListJobs(true)
		removed := 0
		for _, j := range jobs {
			if cronService.RemoveJob(j.ID) {
				removed++
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"removed": removed})
	})

	// Register reload endpoint: desktop GUI POSTs here after the user saves
	// changes on the Config page so the running gateway picks up new model /
	// provider / API key settings without requiring a manual gateway restart.
	healthServer.HandleFunc("/api/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		newCfg, err := internal.LoadConfig()
		if err != nil {
			logger.ErrorCF("gateway", "Reload failed to load config",
				map[string]any{"error": err.Error()})
			http.Error(w, `{"error":"failed to reload config"}`, http.StatusInternalServerError)
			return
		}
		agentLoop.Reload(newCfg)

		// Reload channels so newly-enabled ones (e.g. Telegram) start polling
		// without a gateway restart. Failures are logged, not propagated.
		if err := channelManager.Reload(ctx, newCfg); err != nil {
			logger.ErrorCF("gateway", "Channel reload failed",
				map[string]any{"error": err.Error()})
		}

		// Re-attach the voice transcriber: channelManager.Reload built new
		// channel instances, so the references set up at startup are gone.
		if transcriber != nil {
			if telegramChannel, ok := channelManager.GetChannel("telegram"); ok {
				if tc, ok := telegramChannel.(*channels.TelegramChannel); ok {
					tc.SetTranscriber(transcriber)
				}
			}
			if discordChannel, ok := channelManager.GetChannel("discord"); ok {
				if dc, ok := discordChannel.(*channels.DiscordChannel); ok {
					dc.SetTranscriber(transcriber)
				}
			}
			if slackChannel, ok := channelManager.GetChannel("slack"); ok {
				if sc, ok := slackChannel.(*channels.SlackChannel); ok {
					sc.SetTranscriber(transcriber)
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"reloaded"}`))
	})

	go func() {
		if err := healthServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.ErrorCF("health", "Health server error", map[string]any{"error": err.Error()})
		}
	}()
	fmt.Printf("✓ Health endpoints available at http://%s:%d/health and /ready\n", cfg.Gateway.Host, cfg.Gateway.Port)
	fmt.Printf("✓ Chat API available at http://%s:%d/api/chat\n", cfg.Gateway.Host, cfg.Gateway.Port)
	fmt.Printf("✓ Reload API available at http://%s:%d/api/reload\n", cfg.Gateway.Host, cfg.Gateway.Port)

	go agentLoop.Run(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	fmt.Println("\nShutting down...")
	cancel()
	healthServer.Stop(context.Background())
	deviceService.Stop()
	heartbeatService.Stop()
	cronService.Stop()
	agentLoop.Stop()
	channelManager.StopAll(ctx)
	fmt.Println("✓ Gateway stopped")

	return nil
}

func setupCronTool(
	agentLoop *agent.AgentLoop,
	msgBus *bus.MessageBus,
	workspace string,
	restrict bool,
	execTimeout time.Duration,
	cfg *config.Config,
) *cron.CronService {
	cronStorePath := filepath.Join(workspace, "cron", "jobs.json")

	// Create cron service
	cronService := cron.NewCronService(cronStorePath, nil)

	// Create and register CronTool
	cronTool := tools.NewCronTool(cronService, agentLoop, msgBus, workspace, restrict, execTimeout, cfg)
	agentLoop.RegisterTool(cronTool)

	// Set the onJob handler
	cronService.SetOnJob(func(job *cron.CronJob) (string, error) {
		result := cronTool.ExecuteJob(context.Background(), job)
		return result, nil
	})

	return cronService
}
