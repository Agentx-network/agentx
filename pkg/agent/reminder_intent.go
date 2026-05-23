package agent

import (
	"regexp"
	"strconv"
	"strings"
)

// reminderTrigger matches an explicit request to schedule a timed reminder.
// Covers both directives addressed to the agent ("remind/ping/alert/notify/wake
// me") and the "set / send / schedule / create / make a reminder" phrasings,
// which weak models often produce instead of calling cron.
var reminderTrigger = regexp.MustCompile(
	`(?i)\b(?:` +
		`(?:remind|ping|alert|notify|wake)\s+me` + // "remind me", "ping me", …
		`|(?:set|send|schedule|create|make|add)\s+(?:a|an|another|me\s+a|me\s+an)\s+(?:reminder|alert|ping|alarm|notification)` +
		`)\b`)

// reminderDelay matches a relative delay like "in 5 minutes", "after 30 sec",
// "for 2 hours", "in an hour". Only relative delays are handled
// deterministically; absolute times ("at 5pm", "tomorrow") are left to the model.
var reminderDelay = regexp.MustCompile(
	`(?i)\b(?:in|after|for)\s+(\d+|a|an|one)\s*(second|sec|minute|min|hour|hr)s?(?:\s+from\s+now)?\b`)

// reminderSubject pulls the thing to be reminded about ("...to drink water").
var reminderSubject = regexp.MustCompile(`(?i)\b(?:to|about|that)\s+(.+)$`)

// channelMention finds an explicitly named delivery channel.
var channelMention = regexp.MustCompile(`(?i)\b(telegram|discord|slack|whatsapp)\b`)

// parseReminderIntent deterministically detects "remind/ping me in <time>"
// requests and extracts the delay, subject, and (optional) target channel. It
// is intentionally strict — it only fires when BOTH a reminder trigger and a
// parseable relative delay are present — so it never misfires on ordinary chat.
// Used as a code-level fallback so reminders work even when a weak model fails
// to call the cron tool itself.
func parseReminderIntent(message string) (delaySeconds int, subject, channel string, ok bool) {
	msg := strings.TrimSpace(message)
	if msg == "" || !reminderTrigger.MatchString(msg) {
		return 0, "", "", false
	}
	dm := reminderDelay.FindStringSubmatchIndex(msg)
	if dm == nil {
		return 0, "", "", false
	}
	qty := msg[dm[2]:dm[3]]
	unit := strings.ToLower(msg[dm[4]:dm[5]])

	n := 1
	switch strings.ToLower(qty) {
	case "a", "an", "one":
		n = 1
	default:
		v, err := strconv.Atoi(qty)
		if err != nil || v <= 0 {
			return 0, "", "", false
		}
		n = v
	}
	switch {
	case strings.HasPrefix(unit, "sec"):
		delaySeconds = n
	case strings.HasPrefix(unit, "min"):
		delaySeconds = n * 60
	case strings.HasPrefix(unit, "h"):
		delaySeconds = n * 3600
	}
	if delaySeconds <= 0 || delaySeconds > 7*24*3600 { // cap at 7 days
		return 0, "", "", false
	}

	if m := channelMention.FindStringSubmatch(msg); m != nil {
		channel = strings.ToLower(m[1])
	}

	// Subject: strip the delay phrase, then take the text after "to/about/that".
	stripped := strings.TrimSpace(msg[:dm[0]] + " " + msg[dm[1]:])
	if sm := reminderSubject.FindStringSubmatch(stripped); sm != nil {
		subject = strings.TrimSpace(strings.Trim(sm[1], " ?.!"))
		// Drop a trailing channel mention like "to drink water on telegram".
		subject = strings.TrimSpace(channelMention.ReplaceAllString(subject, ""))
		subject = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(subject), "on"))
	}
	return delaySeconds, subject, channel, true
}

// reminderConfirmationText produces the user-facing one-line confirmation for a
// scheduled reminder. Used by both the deterministic fallback and the override
// path that fires when the model itself called cron successfully — that way the
// reply is short, human, and identical regardless of which path scheduled it,
// instead of the model elaborating with job IDs and follow-up questions.
func reminderConfirmationText(subject string, delaySec int, reqChannel string) string {
	delay := humanizeDelay(delaySec)
	sub := strings.TrimSpace(subject)

	suffix := ""
	if reqChannel != "" {
		// Title-case the channel for display: "telegram" → "Telegram".
		c := strings.TrimSpace(reqChannel)
		if c != "" {
			c = strings.ToUpper(c[:1]) + strings.ToLower(c[1:])
		}
		suffix = " (on " + c + ")"
	}

	if sub == "" {
		return "Done — reminder set for " + delay + " from now" + suffix + "."
	}
	return "Done — I'll remind you to " + sub + " in " + delay + suffix + "."
}

// humanizeDelay renders a second count as a short human phrase.
func humanizeDelay(seconds int) string {
	switch {
	case seconds%3600 == 0 && seconds >= 3600:
		h := seconds / 3600
		if h == 1 {
			return "1 hour"
		}
		return strconv.Itoa(h) + " hours"
	case seconds%60 == 0 && seconds >= 60:
		m := seconds / 60
		if m == 1 {
			return "1 minute"
		}
		return strconv.Itoa(m) + " minutes"
	default:
		if seconds == 1 {
			return "1 second"
		}
		return strconv.Itoa(seconds) + " seconds"
	}
}
