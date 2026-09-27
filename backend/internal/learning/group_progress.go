package learning

import "context"

type GroupProgress struct {
	TasksPerMember  int
	CompletedByUser map[uint64]int
}

// GroupProgressForDates uses the same task identities and completion rules as
// TodayHub. Weekly records are loaded once per period, including carried videos.
func (s *Service) GroupProgressForDates(ctx context.Context, groupID uint64, dates []string) (map[string]GroupProgress, error) {
	out := make(map[string]GroupProgress, len(dates))
	if len(dates) == 0 {
		return out, nil
	}
	settings, err := s.repo.LearningConfig(ctx, groupID)
	if err != nil {
		return nil, err
	}
	weeks, err := s.repo.ListWeeks(ctx, groupID)
	if err != nil {
		return nil, err
	}
	tasksByWeek, err := s.tasksByWeek(ctx, groupID, weeks)
	if err != nil {
		return nil, err
	}
	recordsByPeriod := make(map[string][]TodayRecord)
	for _, date := range dates {
		var week map[string]any
		var tasks []map[string]any
		from, to := date, date
		// ListWeeks is ordered by start date; choose the latest matching week
		// just as CurrentWeek does, including when old schedules overlap.
		for i := len(weeks) - 1; i >= 0; i-- {
			if weeks[i].StartDate <= date && date <= weeks[i].EndDate {
				week = WeekMap(weeks[i])
				tasks = tasksByWeek[weeks[i].ID]
				from, to = weeks[i].StartDate, weeks[i].EndDate
				break
			}
		}
		key := from + "/" + to
		records, ok := recordsByPeriod[key]
		if !ok {
			records, err = s.repo.ListCompletionRecords(ctx, groupID, 0, from, to)
			if err != nil {
				return nil, err
			}
			recordsByPeriod[key] = records
		}
		progress := GroupProgress{
			TasksPerMember:  len(buildTodayTasks(date, week, tasks, settings, nil)),
			CompletedByUser: make(map[uint64]int),
		}
		for _, item := range buildGroupTaskCompletions(date, week, tasks, settings, records) {
			progress.CompletedByUser[item.UserID]++
		}
		out[date] = progress
	}
	return out, nil
}
