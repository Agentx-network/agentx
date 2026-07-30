package agent

import "testing"

func TestShouldAutoSkillSearch(t *testing.T) {
	// Capability-gap deflections: a skill could plausibly fill the gap, so we
	// want the fallback to fire and run find_skills.
	shouldFire := []string{
		"I'm sorry, I can't directly compress videos. My current tools don't have that capability.",
		"I don't have that capability right now.",
		"That's outside my current capabilities, unfortunately.",
		"I don't have a tool for editing PDFs.",
		"I'm not equipped to do that at the moment.",
		"I don't have the ability to convert audio files.",
	}
	for _, msg := range shouldFire {
		if !shouldAutoSkillSearch(msg) {
			t.Errorf("expected skill-search to fire for reply: %q", msg)
		}
	}

	// Real answers and bare policy refusals: no missing-tool signal, so the
	// fallback must stay quiet (a find_skills dump would just confuse the user).
	shouldNotFire := []string{
		"Sure! Here's a summary of the article you asked about.",
		"I can't help with that request.",
		"The current price of Bitcoin is $65,000.",
		"",
		"I compressed the video and saved it to your workspace.",
	}
	for _, msg := range shouldNotFire {
		if shouldAutoSkillSearch(msg) {
			t.Errorf("expected skill-search NOT to fire for reply: %q", msg)
		}
	}
}
