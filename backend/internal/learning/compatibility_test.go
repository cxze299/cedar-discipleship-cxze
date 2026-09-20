package learning

import "testing"

func TestSeparateDailyCompletion(t *testing.T) {
	settings := map[string]any{"task_sections": map[string]any{"daily": map[string]any{
		"checkin_mode": "separate",
		"devotion":     map[string]any{"enabled": true},
		"scripture":    map[string]any{"enabled": true, "type": "checkin"},
	}}}
	records := []TodayRecord{{ID: 1, UserID: 2, TaskType: "daily_scripture", LogicalDate: "2026-09-20"}}
	tasks := buildTodayTasks("2026-09-20", nil, nil, settings, records)
	if len(tasks) != 2 || tasks[0].Type != "daily_devotion" || tasks[0].Completed ||
		tasks[1].Type != "daily_scripture" || !tasks[1].Completed {
		t.Fatalf("independent daily tasks = %+v", tasks)
	}
	items := buildGroupTaskCompletions("2026-09-20", nil, nil, settings, records)
	if len(items) != 1 || items[0].TaskType != "daily_scripture" {
		t.Fatalf("group completions = %+v", items)
	}
	if DailyTaskTypeEnabled(map[string]any{}, "daily_scripture") {
		t.Fatal("combined mode must not accept a separate scripture record")
	}
	settings["task_sections"].(map[string]any)["daily"].(map[string]any)["scripture"] = map[string]any{"enabled": false}
	if DailyTaskTypeEnabled(settings, "daily_scripture") {
		t.Fatal("disabled scripture must reject checkins")
	}
}

func TestAggregateWeekRetainsContentWithoutCompletingBooks(t *testing.T) {
	weekID := uint64(7)
	input := WeekInput{
		Title: "本周两个主题", WeeklyCheckin: true, BookEnabled: true,
		Readings: []TaskBinding{{Title: "第一章", URL: "https://example.org/chapter.htm"}},
	}
	if title := WeekTitle(input); title != input.Title {
		t.Fatalf("aggregate week title = %q", title)
	}
	drafts := BuildTaskDrafts(input, "")
	var aggregate, book *TaskDraft
	for i := range drafts {
		switch drafts[i].TaskType {
		case "weekly_checkin":
			aggregate = &drafts[i]
		case "weekly_book":
			book = &drafts[i]
		}
	}
	if aggregate == nil || book == nil || !book.Optional {
		t.Fatalf("drafts must retain optional content and aggregate completion: %+v", drafts)
	}
	raw := []map[string]any{
		{"id": uint64(11), "task_type": "weekly_checkin", "title": input.Title},
		{"id": uint64(12), "task_type": "weekly_book", "title": "第一章"},
	}
	tasks := buildTodayTasks("2026-09-20", map[string]any{"id": weekID, "book_enabled": false},
		raw, map[string]any{}, []TodayRecord{
			{ID: 1, TaskType: "weekly_book", WeekID: &weekID, Part: "第一章", LogicalDate: "2026-09-20"},
		})
	if len(tasks) != 2 || tasks[1].Type != "weekly_checkin" || tasks[1].Completed {
		t.Fatalf("aggregate task must be independent of reading toggle and book records: %+v", tasks)
	}
	record := TodayRecord{ID: 2, TaskType: "weekly_checkin", WeekID: &weekID, LogicalDate: "2026-09-18"}
	if matchingTodayRecord(tasks[1], []TodayRecord{record}, "2026-09-20") == nil {
		t.Fatal("aggregate completion should match within its week")
	}
	if matchingTodayRecord(TodayTaskVO{Type: "weekly_book", WeekID: weekID}, []TodayRecord{record}, "2026-09-20") != nil {
		t.Fatal("aggregate completion must not satisfy an individual book")
	}
}

func TestLegacyContentBindings(t *testing.T) {
	if got := InferTaskBindingType("weekly_book", "https://example.org/chapter.htm?v=1#part", ""); got != "iframe" {
		t.Fatalf("external HTML binding = %q", got)
	}
	if drafts := BuildTaskDrafts(WeekInput{VideoEnabled: true, VerseEnabled: true}, ""); len(drafts) != 0 {
		t.Fatalf("empty content created required tasks: %+v", drafts)
	}
}
