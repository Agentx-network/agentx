package agent

import (
	"regexp"
	"strconv"
	"strings"
)

// reminderTrigger matches an explicit request to be reminded/pinged.
var reminderTrigger = regexp.MustCompile(`(?i)\b(remind|ping|alert|notify|wake)\s+me\b`)

// reminderDelay matches a relative delay like "in 5 minutes", "in 30 sec",
// "in an hour". Only relative delays are handled deterministically; absolute
// times ("at 5pm", "tomorrow") are left to the model.
var reminderDelay = regexp.MustCompile(`(?i)\bin\s+(\d+|a|an|one)\s*(second|sec|minute|min|hour|hr)s?\b`)

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
