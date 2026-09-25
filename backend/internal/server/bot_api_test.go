package server

import "testing"

func TestBotTaskTypeSupportsExistingCheckinTypes(t *testing.T) {
	for input, want := range map[string]string{
		"daily_devotion":  "daily_devotion",
		"每日读经":          "daily_scripture",
		"daily_scripture": "daily_scripture",
		"周任务":            "weekly_checkin",
		"weekly_checkin":  "weekly_checkin",
		"weekly_book":     "weekly_book",
	} {
		if got := botTaskType(input); got != want {
			t.Errorf("botTaskType(%q) = %q, want %q", input, got, want)
		}
	}
}
