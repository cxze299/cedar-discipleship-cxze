package learning

func SeparateDailyCheckins(settings map[string]any) bool {
	return nestedString(settings, []string{"task_sections", "daily", "checkin_mode"}, "combined") == "separate"
}

func DailyTaskTypeEnabled(settings map[string]any, taskType string) bool {
	if !SeparateDailyCheckins(settings) {
		return taskType == "daily_devotion" && DailyTaskEnabled(settings)
	}
	component := ""
	switch taskType {
	case "daily_devotion":
		component = "devotion"
	case "daily_scripture":
		component = "scripture"
	default:
		return false
	}
	config, exists := nestedMap(settings, "task_sections", "daily", component)
	return !exists || mapBool(config, "enabled", true)
}

func dailyTasks(settings map[string]any) []TodayTaskVO {
	var tasks []TodayTaskVO
	for _, taskType := range []string{"daily_devotion", "daily_scripture"} {
		if !DailyTaskTypeEnabled(settings, taskType) {
			continue
		}
		title := nestedString(settings, []string{"task_sections", "daily", "label"}, "每日灵修")
		summary := "今天的灵修与读经"
		if SeparateDailyCheckins(settings) {
			title = nestedString(settings, []string{"task_sections", "daily", "devotion", "title"}, "每日灵修")
			if taskType == "daily_scripture" {
				title = nestedString(settings, []string{"task_sections", "daily", "scripture", "label"}, "每日读经")
			}
			summary = title
		}
		tasks = append(tasks, TodayTaskVO{
			ID: taskType, Type: taskType, Kind: todayTaskKind(taskType),
			Title: title, Detail: title, Summary: summary,
			Required: true, Status: "pending",
		})
	}
	return tasks
}

func hasAggregateWeeklyTask(tasks []map[string]any) bool {
	for _, task := range tasks {
		if asString(task["task_type"]) == "weekly_checkin" && mapBool(task, "enabled", true) {
			return true
		}
	}
	return false
}
