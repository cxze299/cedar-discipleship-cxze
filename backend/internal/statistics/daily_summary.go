package statistics

import (
	"context"

	"agp/backend/internal/learning"
)

type DailySummary struct {
	LogicalDate    string
	CheckedMembers int
	TotalCheckins  int
	DevotionCount  int
	ScriptureCount int
	WeeklyCount    int
	BookCount      int
	VideoCount     int
	VerseCount     int
	OutlineCount   int
	ExpectedTasks  int
	CompletedTasks int
}

func (s *Service) DailySummaries(ctx context.Context, groupID uint64, tasks *learning.Service) (int, []DailySummary, error) {
	members, err := s.repo.Members(ctx, groupID)
	if err != nil {
		return 0, nil, err
	}
	items, err := s.repo.DailyEvents(ctx, groupID)
	if err != nil {
		return 0, nil, err
	}
	dates := make([]string, 0, len(items))
	for _, item := range items {
		dates = append(dates, item.LogicalDate)
	}
	progress, err := tasks.GroupProgressForDates(ctx, groupID, dates)
	if err != nil {
		return 0, nil, err
	}
	for i := range items {
		state := progress[items[i].LogicalDate]
		items[i].ExpectedTasks = len(members) * state.TasksPerMember
		for _, member := range members {
			items[i].CompletedTasks += state.CompletedByUser[member.UserID]
		}
	}
	return len(members), items, nil
}
