package channels

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/Agentx-network/agentx/pkg/bus"
	"github.com/Agentx-network/agentx/pkg/config"
	"github.com/Agentx-network/agentx/pkg/logger"
	"github.com/Agentx-network/agentx/pkg/utils"
	"github.com/Agentx-network/agentx/pkg/voice"
)

var (
	reHeading    = regexp.MustCompile(`^#{1,6}\s+(.+)$`)
	reBlockquote = regexp.MustCompile(`^>\s*(.*)$`)
	reLink       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reBoldStar   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reBoldUnder  = regexp.MustCompile(`__(.+?)__`)
	reItalic     = regexp.MustCompile(`_([^_]+)_`)
	reStrike     = regexp.MustCompile(`~~(.+?)~~`)
	reListItem   = regexp.MustCompile(`^[-*]\s+`)
	reCodeBlock  = regexp.MustCompile("```[\\w]*\\n?([\\s\\S]*?)```")
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	reTableSep   = regexp.MustCompile(`^\|[-:\s|]+\|$`)
	reTableRow   = regexp.MustCompile(`^\|(.+)\|$`)
	reHorizRule  = regexp.MustCompile(`^---+$`)
)

type TelegramChannel struct {
	*BaseChannel
	bot             *telego.Bot
	commands        TelegramCommander
	config          *config.Config
	chatIDs         map[string]int64
	transcriber     *voice.GroqTranscriber
	placeholders    sync.Map // chatID -> messageID
	stopThinking    sync.Map // chatID -> thinkingCancel
	streamBuffers   sync.Map // chatID -> *streamBuffer
	streamMsgIDs    sync.Map // chatID -> int (message ID for progressive edits)
}

// streamBuffer accumulates stream deltas for debounced Telegram edits.
type streamBuffer struct {
	mu      sync.Mutex
	content strings.Builder
	dirty   bool
}

type thinkingCancel struct {
	fn context.CancelFunc
}

func (c *thinkingCancel) Cancel() {
	if c != nil && c.fn != nil {
		c.fn()
	}
}

func NewTelegramChannel(cfg *config.Config, bus *bus.MessageBus) (*TelegramChannel, error) {
	var opts []telego.BotOption
	telegramCfg := cfg.Channels.Telegram

	if telegramCfg.Proxy != "" {
		proxyURL, parseErr := url.Parse(telegramCfg.Proxy)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", telegramCfg.Proxy, parseErr)
		}
		opts = append(opts, telego.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}))
	} else if os.Getenv("HTTP_PROXY") != "" || os.Getenv("HTTPS_PROXY") != "" {
		// Use environment proxy if configured
		opts = append(opts, telego.WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		}))
	}

	bot, err := telego.NewBot(telegramCfg.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	base := NewBaseChannel("telegram", telegramCfg, bus, telegramCfg.AllowFrom)

	return &TelegramChannel{
		BaseChannel:  base,
		commands:     NewTelegramCommands(bot, cfg),
		bot:          bot,
		config:       cfg,
		chatIDs:      make(map[string]int64),
		transcriber:  nil,
		placeholders: sync.Map{},
		stopThinking: sync.Map{},
	}, nil
}

func (c *TelegramChannel) SetTranscriber(transcriber *voice.GroqTranscriber) {
	c.transcriber = transcriber
}

func (c *TelegramChannel) Start(ctx context.Context) error {
	logger.InfoC("telegram", "Starting Telegram bot (polling mode)...")

	updates, err := c.bot.UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{
		Timeout: 30,
	})
	if err != nil {
		return fmt.Errorf("failed to start long polling: %w", err)
	}

	bh, err := telegohandler.NewBotHandler(c.bot, updates)
	if err != nil {
		return fmt.Errorf("failed to create bot handler: %w", err)
	}

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		c.commands.Help(ctx, message)
		return nil
	}, th.CommandEqual("help"))
	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.commands.Start(ctx, message)
	}, th.CommandEqual("start"))

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.commands.Show(ctx, message)
	}, th.CommandEqual("show"))

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.commands.List(ctx, message)
	}, th.CommandEqual("list"))

	bh.HandleMessage(func(ctx *th.Context, message telego.Message) error {
		return c.handleMessage(ctx, &message)
	}, th.AnyMessage())

	c.setRunning(true)
	logger.InfoCF("telegram", "Telegram bot connected", map[string]any{
		"username": c.bot.Username(),
	})

	go bh.Start()

	go func() {
		<-ctx.Done()
		bh.Stop()
	}()

	return nil
}

// StartStreamConsumer starts a goroutine that consumes stream deltas and
// progressively updates Telegram messages with debounced edits.
// It reuses the "Thinking..." placeholder message instead of creating new ones.
func (c *TelegramChannel) StartStreamConsumer(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Flush all dirty stream buffers
				c.streamBuffers.Range(func(key, value any) bool {
					chatIDStr := key.(string)
					buf := value.(*streamBuffer)

					buf.mu.Lock()
					if !buf.dirty {
						buf.mu.Unlock()
						return true
					}
					content := buf.content.String()
					buf.dirty = false
					buf.mu.Unlock()

					if content == "" {
						return true
					}

					chatID, err := parseChatID(chatIDStr)
					if err != nil {
						return true
					}

					// Truncate streaming preview to avoid hitting Telegram limits mid-stream.
					// The full content will be sent via Send() when done.
					displayContent := content
					if len(displayContent) > 3500 {
						displayContent = displayContent[:3500] + "\n\n..."
					}

					htmlContent := markdownToTelegramHTML(displayContent)

					// Try to reuse existing stream message, or claim the placeholder
					if msgIDVal, ok := c.streamMsgIDs.Load(chatIDStr); ok {
						msgID := msgIDVal.(int)
						editMsg := tu.EditMessageText(tu.ID(chatID), msgID, htmlContent)
						editMsg.ParseMode = telego.ModeHTML
						if _, err := c.bot.EditMessageText(ctx, editMsg); err != nil {
							logger.DebugCF("telegram", "Stream edit failed", map[string]any{
								"error": err.Error(),
							})
						}
					} else if pID, ok := c.placeholders.Load(chatIDStr); ok {
						// Claim the placeholder for streaming
						msgID := pID.(int)
						c.placeholders.Delete(chatIDStr)
						c.streamMsgIDs.Store(chatIDStr, msgID)
						editMsg := tu.EditMessageText(tu.ID(chatID), msgID, htmlContent)
						editMsg.ParseMode = telego.ModeHTML
						if _, err := c.bot.EditMessageText(ctx, editMsg); err != nil {
							logger.DebugCF("telegram", "Stream placeholder edit failed", map[string]any{
								"error": err.Error(),
							})
						}
					} else {
						// No placeholder available, create a new message
						tgMsg := tu.Message(tu.ID(chatID), htmlContent)
						tgMsg.ParseMode = telego.ModeHTML
						sent, err := c.bot.SendMessage(ctx, tgMsg)
						if err == nil {
							c.streamMsgIDs.Store(chatIDStr, sent.MessageID)
						}
					}

					return true
				})
			}
		}
	}()
}

// HandleStreamDelta processes a single stream delta event.
func (c *TelegramChannel) HandleStreamDelta(delta bus.StreamDelta) {
	if delta.Done {
		// Keep streamMsgIDs so Send() can edit the stream message with the final content.
		// Only clean up the buffer.
		c.streamBuffers.Delete(delta.ChatID)
		return
	}

	val, _ := c.streamBuffers.LoadOrStore(delta.ChatID, &streamBuffer{})
	buf := val.(*streamBuffer)
	buf.mu.Lock()
	buf.content.WriteString(delta.Delta)
	buf.dirty = true
	buf.mu.Unlock()

	// Stop thinking animation on first delta
	if stop, ok := c.stopThinking.Load(delta.ChatID); ok {
		if cf, ok := stop.(*thinkingCancel); ok && cf != nil {
			cf.Cancel()
		}
		c.stopThinking.Delete(delta.ChatID)
	}
}

func (c *TelegramChannel) Stop(ctx context.Context) error {
	logger.InfoC("telegram", "Stopping Telegram bot...")
	c.setRunning(false)
	return nil
}

// telegramMaxMessageLength is Telegram's limit for a single message.
const telegramMaxMessageLength = 4096

func (c *TelegramChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("telegram bot not running")
	}

	chatID, err := parseChatID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	// Stop thinking animation
	if stop, ok := c.stopThinking.Load(msg.ChatID); ok {
		if cf, ok := stop.(*thinkingCancel); ok && cf != nil {
			cf.Cancel()
		}
		c.stopThinking.Delete(msg.ChatID)
	}

	htmlContent := markdownToTelegramHTML(msg.Content)

	// Find a message to edit: prefer stream message, then placeholder
	var editMsgID int
	if smID, ok := c.streamMsgIDs.LoadAndDelete(msg.ChatID); ok {
		editMsgID = smID.(int)
		// Also clean up placeholder since we're using the stream message
		c.placeholders.Delete(msg.ChatID)
	} else if pID, ok := c.placeholders.LoadAndDelete(msg.ChatID); ok {
		editMsgID = pID.(int)
	}

	// Split into chunks if content exceeds Telegram's limit
	chunks := splitTelegramMessage(htmlContent)

	for i, chunk := range chunks {
		if i == 0 && editMsgID != 0 {
			// Edit the existing message with the first chunk
			editMsg := tu.EditMessageText(tu.ID(chatID), editMsgID, chunk)
			editMsg.ParseMode = telego.ModeHTML
			if _, err = c.bot.EditMessageText(ctx, editMsg); err == nil {
				continue
			}
			// Edit failed — fall through to send as new message
		}
		if err := c.sendChunk(ctx, chatID, chunk); err != nil {
			return err
		}
	}

	return nil
}

// sendChunk sends a single message chunk, falling back to plain text if HTML fails.
func (c *TelegramChannel) sendChunk(ctx context.Context, chatID int64, htmlContent string) error {
	tgMsg := tu.Message(tu.ID(chatID), htmlContent)
	tgMsg.ParseMode = telego.ModeHTML

	if _, err := c.bot.SendMessage(ctx, tgMsg); err != nil {
		logger.DebugCF("telegram", "HTML send failed, falling back to plain text", map[string]any{
			"error": err.Error(),
		})
		tgMsg.ParseMode = ""
		_, err = c.bot.SendMessage(ctx, tgMsg)
		return err
	}
	return nil
}

// splitTelegramMessage splits HTML content into chunks that fit within Telegram's
// message length limit. It splits on paragraph boundaries (\n\n) to keep
// formatting intact.
func splitTelegramMessage(html string) []string {
	if len(html) <= telegramMaxMessageLength {
		return []string{html}
	}

	var chunks []string
	remaining := html

	for len(remaining) > 0 {
		if len(remaining) <= telegramMaxMessageLength {
			chunks = append(chunks, remaining)
			break
		}

		// Find a good split point — prefer double newline, then single newline
		cutAt := telegramMaxMessageLength
		if idx := strings.LastIndex(remaining[:cutAt], "\n\n"); idx > cutAt/2 {
			cutAt = idx
		} else if idx := strings.LastIndex(remaining[:cutAt], "\n"); idx > cutAt/2 {
			cutAt = idx
		}

		chunk := strings.TrimSpace(remaining[:cutAt])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		remaining = strings.TrimSpace(remaining[cutAt:])
	}

	return chunks
}

func (c *TelegramChannel) handleMessage(ctx context.Context, message *telego.Message) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}

	user := message.From
	if user == nil {
		return fmt.Errorf("message sender (user) is nil")
	}

	senderID := fmt.Sprintf("%d", user.ID)
	if user.Username != "" {
		senderID = fmt.Sprintf("%d|%s", user.ID, user.Username)
	}

	// check allowlist to avoid downloading attachments for rejected users
	if !c.IsAllowed(senderID) {
		logger.DebugCF("telegram", "Message rejected by allowlist", map[string]any{
			"user_id": senderID,
		})
		return nil
	}

	chatID := message.Chat.ID
	c.chatIDs[senderID] = chatID

	content := ""
	mediaPaths := []string{}
	localFiles := []string{} // track local files that need cleanup

	// ensure temp files are cleaned up when function returns
	defer func() {
		for _, file := range localFiles {
			if err := os.Remove(file); err != nil {
				logger.DebugCF("telegram", "Failed to cleanup temp file", map[string]any{
					"file":  file,
					"error": err.Error(),
				})
			}
		}
	}()

	if message.Text != "" {
		content += message.Text
	}

	if message.Caption != "" {
		if content != "" {
			content += "\n"
		}
		content += message.Caption
	}

	if len(message.Photo) > 0 {
		photo := message.Photo[len(message.Photo)-1]
		photoPath := c.downloadPhoto(ctx, photo.FileID)
		if photoPath != "" {
			localFiles = append(localFiles, photoPath)
			mediaPaths = append(mediaPaths, photoPath)
			if content != "" {
				content += "\n"
			}
			content += "[image: photo]"
		}
	}

	if message.Voice != nil {
		voicePath := c.downloadFile(ctx, message.Voice.FileID, ".ogg")
		if voicePath != "" {
			localFiles = append(localFiles, voicePath)
			mediaPaths = append(mediaPaths, voicePath)

			var transcribedText string
			if c.transcriber != nil && c.transcriber.IsAvailable() {
				transcriberCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()

				result, err := c.transcriber.Transcribe(transcriberCtx, voicePath)
				if err != nil {
					logger.ErrorCF("telegram", "Voice transcription failed", map[string]any{
						"error": err.Error(),
						"path":  voicePath,
					})
					transcribedText = "[voice (transcription failed)]"
				} else {
					transcribedText = fmt.Sprintf("[voice transcription: %s]", result.Text)
					logger.InfoCF("telegram", "Voice transcribed successfully", map[string]any{
						"text": result.Text,
					})
				}
			} else {
				transcribedText = "[voice]"
			}

			if content != "" {
				content += "\n"
			}
			content += transcribedText
		}
	}

	if message.Audio != nil {
		audioPath := c.downloadFile(ctx, message.Audio.FileID, ".mp3")
		if audioPath != "" {
			localFiles = append(localFiles, audioPath)
			mediaPaths = append(mediaPaths, audioPath)
			if content != "" {
				content += "\n"
			}
			content += "[audio]"
		}
	}

	if message.Document != nil {
		docPath := c.downloadFile(ctx, message.Document.FileID, "")
		if docPath != "" {
			localFiles = append(localFiles, docPath)
			mediaPaths = append(mediaPaths, docPath)
			if content != "" {
				content += "\n"
			}
			content += "[file]"
		}
	}

	if content == "" {
		content = "[empty message]"
	}

	logger.DebugCF("telegram", "Received message", map[string]any{
		"sender_id": senderID,
		"chat_id":   fmt.Sprintf("%d", chatID),
		"preview":   utils.Truncate(content, 50),
	})

	// Thinking indicator
	err := c.bot.SendChatAction(ctx, tu.ChatAction(tu.ID(chatID), telego.ChatActionTyping))
	if err != nil {
		logger.ErrorCF("telegram", "Failed to send chat action", map[string]any{
			"error": err.Error(),
		})
	}

	// Stop any previous thinking animation
	chatIDStr := fmt.Sprintf("%d", chatID)
	if prevStop, ok := c.stopThinking.Load(chatIDStr); ok {
		if cf, ok := prevStop.(*thinkingCancel); ok && cf != nil {
			cf.Cancel()
		}
	}

	// Create cancel function for thinking state
	_, thinkCancel := context.WithTimeout(ctx, 5*time.Minute)
	c.stopThinking.Store(chatIDStr, &thinkingCancel{fn: thinkCancel})

	pMsg, err := c.bot.SendMessage(ctx, tu.Message(tu.ID(chatID), "Thinking... 💭"))
	if err == nil {
		pID := pMsg.MessageID
		c.placeholders.Store(chatIDStr, pID)
	}

	peerKind := "direct"
	peerID := fmt.Sprintf("%d", user.ID)
	if message.Chat.Type != "private" {
		peerKind = "group"
		peerID = fmt.Sprintf("%d", chatID)
	}

	metadata := map[string]string{
		"message_id": fmt.Sprintf("%d", message.MessageID),
		"user_id":    fmt.Sprintf("%d", user.ID),
		"username":   user.Username,
		"first_name": user.FirstName,
		"is_group":   fmt.Sprintf("%t", message.Chat.Type != "private"),
		"peer_kind":  peerKind,
		"peer_id":    peerID,
	}

	c.HandleMessage(fmt.Sprintf("%d", user.ID), fmt.Sprintf("%d", chatID), content, mediaPaths, metadata)
	return nil
}

func (c *TelegramChannel) downloadPhoto(ctx context.Context, fileID string) string {
	file, err := c.bot.GetFile(ctx, &telego.GetFileParams{FileID: fileID})
	if err != nil {
		logger.ErrorCF("telegram", "Failed to get photo file", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	return c.downloadFileWithInfo(file, ".jpg")
}

func (c *TelegramChannel) downloadFileWithInfo(file *telego.File, ext string) string {
	if file.FilePath == "" {
		return ""
	}

	url := c.bot.FileDownloadURL(file.FilePath)
	logger.DebugCF("telegram", "File URL", map[string]any{"url": url})

	// Use FilePath as filename for better identification
	filename := file.FilePath + ext
	return utils.DownloadFile(url, filename, utils.DownloadOptions{
		LoggerPrefix: "telegram",
	})
}

func (c *TelegramChannel) downloadFile(ctx context.Context, fileID, ext string) string {
	file, err := c.bot.GetFile(ctx, &telego.GetFileParams{FileID: fileID})
	if err != nil {
		logger.ErrorCF("telegram", "Failed to get file", map[string]any{
			"error": err.Error(),
		})
		return ""
	}

	return c.downloadFileWithInfo(file, ext)
}

func parseChatID(chatIDStr string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(chatIDStr, "%d", &id)
	return id, err
}

func markdownToTelegramHTML(text string) string {
	if text == "" {
		return ""
	}

	codeBlocks := extractCodeBlocks(text)
	text = codeBlocks.text

	inlineCodes := extractInlineCodes(text)
	text = inlineCodes.text

	// Convert markdown tables before other processing
	text = convertMarkdownTables(text)

	// Remove horizontal rules
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if reHorizRule.MatchString(strings.TrimSpace(line)) {
			lines[i] = ""
		}
	}
	text = strings.Join(lines, "\n")

	text = reHeading.ReplaceAllString(text, "$1")

	text = reBlockquote.ReplaceAllString(text, "$1")

	text = escapeHTML(text)

	text = reLink.ReplaceAllString(text, `<a href="$2">$1</a>`)

	text = reBoldStar.ReplaceAllString(text, "<b>$1</b>")

	text = reBoldUnder.ReplaceAllString(text, "<b>$1</b>")

	text = reItalic.ReplaceAllStringFunc(text, func(s string) string {
		match := reItalic.FindStringSubmatch(s)
		if len(match) < 2 {
			return s
		}
		return "<i>" + match[1] + "</i>"
	})

	text = reStrike.ReplaceAllString(text, "<s>$1</s>")

	text = reListItem.ReplaceAllString(text, "• ")

	for i, code := range inlineCodes.codes {
		escaped := escapeHTML(code)
		text = strings.ReplaceAll(text, fmt.Sprintf("\x00IC%d\x00", i), fmt.Sprintf("<code>%s</code>", escaped))
	}

	for i, code := range codeBlocks.codes {
		escaped := escapeHTML(code)
		text = strings.ReplaceAll(
			text,
			fmt.Sprintf("\x00CB%d\x00", i),
			fmt.Sprintf("<pre><code>%s</code></pre>", escaped),
		)
	}

	return text
}

// convertMarkdownTables detects markdown tables and converts them to a readable
// list format since Telegram doesn't support HTML tables.
// Each data row becomes a formatted entry with "header: value" pairs.
func convertMarkdownTables(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	i := 0

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Check if this line starts a table (pipe-delimited row)
		if !reTableRow.MatchString(line) {
			result = append(result, lines[i])
			i++
			continue
		}

		// Parse the header row
		headers := parseTableRow(line)
		if len(headers) == 0 {
			result = append(result, lines[i])
			i++
			continue
		}
		i++

		// Skip separator row (|---|---|)
		if i < len(lines) && reTableSep.MatchString(strings.TrimSpace(lines[i])) {
			i++
		}

		// Process data rows
		rowNum := 0
		for i < len(lines) {
			rowLine := strings.TrimSpace(lines[i])
			if !reTableRow.MatchString(rowLine) {
				break
			}

			cells := parseTableRow(rowLine)
			if len(cells) == 0 {
				break
			}

			if rowNum > 0 {
				result = append(result, "")
			}

			// Build a compact entry: use first non-index column as the title,
			// then list remaining columns as "header: value"
			entry := formatTableRow(headers, cells)
			result = append(result, entry)
			rowNum++
			i++
		}

		// Add spacing after table
		result = append(result, "")
	}

	return strings.Join(result, "\n")
}

// parseTableRow splits a markdown table row into trimmed cell values.
func parseTableRow(line string) []string {
	match := reTableRow.FindStringSubmatch(line)
	if len(match) < 2 {
		return nil
	}
	parts := strings.Split(match[1], "|")
	cells := make([]string, 0, len(parts))
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// formatTableRow formats a single table data row as a readable text block.
func formatTableRow(headers, cells []string) string {
	// Find the "main" column: skip columns that look like a row index (#, No, etc.)
	mainIdx := -1
	for j, h := range headers {
		hl := strings.ToLower(h)
		if hl == "#" || hl == "no" || hl == "no." || hl == "idx" || hl == "" {
			continue
		}
		mainIdx = j
		break
	}

	var parts []string

	if mainIdx >= 0 && mainIdx < len(cells) {
		// Use the main column value as title line
		parts = append(parts, cells[mainIdx])

		// Append remaining columns as "header: value" on the next line
		var details []string
		for j := 0; j < len(headers) && j < len(cells); j++ {
			if j == mainIdx || cells[j] == "" {
				continue
			}
			h := headers[j]
			hl := strings.ToLower(h)
			if hl == "#" || hl == "no" || hl == "no." || hl == "idx" {
				continue
			}
			details = append(details, h+": "+cells[j])
		}
		if len(details) > 0 {
			parts = append(parts, "  "+strings.Join(details, " | "))
		}
	} else {
		// Fallback: just join all cells
		parts = append(parts, strings.Join(cells, " | "))
	}

	return strings.Join(parts, "\n")
}

type codeBlockMatch struct {
	text  string
	codes []string
}

func extractCodeBlocks(text string) codeBlockMatch {
	matches := reCodeBlock.FindAllStringSubmatch(text, -1)

	codes := make([]string, 0, len(matches))
	for _, match := range matches {
		codes = append(codes, match[1])
	}

	i := 0
	text = reCodeBlock.ReplaceAllStringFunc(text, func(m string) string {
		placeholder := fmt.Sprintf("\x00CB%d\x00", i)
		i++
		return placeholder
	})

	return codeBlockMatch{text: text, codes: codes}
}

type inlineCodeMatch struct {
	text  string
	codes []string
}

func extractInlineCodes(text string) inlineCodeMatch {
	matches := reInlineCode.FindAllStringSubmatch(text, -1)

	codes := make([]string, 0, len(matches))
	for _, match := range matches {
		codes = append(codes, match[1])
	}

	i := 0
	text = reInlineCode.ReplaceAllStringFunc(text, func(m string) string {
		placeholder := fmt.Sprintf("\x00IC%d\x00", i)
		i++
		return placeholder
	})

	return inlineCodeMatch{text: text, codes: codes}
}

func escapeHTML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}
