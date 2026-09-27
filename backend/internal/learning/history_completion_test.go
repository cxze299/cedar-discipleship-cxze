package learning

import "testing"

func TestDailyTaskTypeEnabledOnDate(t *testing.T) {
	for _, mode := range []string{"combined", "separate"} {
		for _, component := range []string{"devotion", "scripture"} {
			for _, tc := range []struct {
				name, date             string
				custom, disabled, want bool
			}{
				{name: "before first schedule", date: "2026-05-26"},
				{name: "historical schedule", date: "2026-09-06", want: true},
				{name: "current schedule", date: "2026-09-07", want: true},
				{name: "historical custom plan", date: "2026-09-06", custom: true, want: true},
				{name: "historical custom gap", date: "2026-09-05", custom: true},
				{name: "disabled remains disabled", date: "2026-09-06", disabled: true},
			} {
				if component == "scripture" && tc.custom {
					continue
				}
				t.Run(mode+"/"+component+"/"+tc.name, func(t *testing.T) {
					current := map[string]any{"enabled": !tc.disabled, "start_date": "2026-09-07"}
					previous := map[string]any{"enabled": true, "start_date": "2026-05-27"}
					if component == "devotion" {
						// numbered_start_date takes precedence over legacy start_date.
						current["numbered_start_date"] = "2026-09-07"
						current["start_date"] = "2026-01-01"
						previous["numbered_start_date"] = "2026-05-27"
					}
					if tc.custom {
						previous["plan_mode"] = "custom"
						previous["plans"] = []any{map[string]any{"date": "2026-09-06", "title": "历史标题"}}
					}
					current["schedule_history"] = []any{"invalid", previous}
					daily := map[string]any{
						"checkin_mode": mode,
						"devotion":     map[string]any{"enabled": false},
						"scripture":    map[string]any{"enabled": false},
					}
					daily[component] = current
					settings := map[string]any{"task_sections": map[string]any{"daily": daily}}
					taskType := "daily_devotion"
					if mode == "separate" && component == "scripture" {
						taskType = "daily_scripture"
					}
					if got := DailyTaskTypeEnabledOnDate(settings, taskType, tc.date); got != tc.want {
						t.Errorf("checkin admission=%v want=%v", got, tc.want)
					}
					records := []TodayRecord{{ID: 1, UserID: 1, LogicalDate: tc.date, TaskType: taskType}}
					tasks := buildTodayTasks(tc.date, nil, nil, settings, records)
					completions := buildGroupTaskCompletions(tc.date, nil, nil, settings, records)
					if !tc.want {
						if len(tasks) != 0 || len(completions) != 0 {
							t.Fatalf("disabled date exposed tasks=%+v completions=%+v", tasks, completions)
						}
						return
					}
					if len(tasks) != 1 || !tasks[0].Completed || len(completions) != 1 {
						t.Fatalf("history is missing: tasks=%+v completions=%+v", tasks, completions)
					}
					if tc.custom && tasks[0].Title != "历史标题" {
						t.Errorf("historical custom title=%q", tasks[0].Title)
					}
					if current["numbered_start_date"] == previous["numbered_start_date"] && component == "devotion" {
						t.Fatal("history resolution mutated current schedule")
					}
				})
			}
		}
	}
}
