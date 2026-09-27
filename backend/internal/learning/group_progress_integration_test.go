//go:build integration

package learning

import (
	"context"
	"testing"
	"time"

	"agp/backend/internal/testdb"
)

type progressQueryRepository struct {
	Repository
	recordQueries int
	taskQueries   int
}

func (r *progressQueryRepository) ListCompletionRecords(ctx context.Context, groupID, userID uint64, from, to string) ([]TodayRecord, error) {
	r.recordQueries++
	return r.Repository.ListCompletionRecords(ctx, groupID, userID, from, to)
}

func (r *progressQueryRepository) ListTasksForWeeks(ctx context.Context, groupID uint64, ids []uint64) ([]Task, error) {
	r.taskQueries++
	return r.Repository.ListTasksForWeeks(ctx, groupID, ids)
}

func TestGroupProgressForDates(t *testing.T) {
	db := testdb.Open(t)
	testdb.Exec(t, db, `INSERT INTO study_weeks(id,group_id,start_date,end_date,created_at,updated_at)
		VALUES (1,1,'2026-09-14','2026-09-20',NOW(),NOW()),(2,1,'2026-09-21','2026-09-27',NOW(),NOW()),
		       (3,2,'2026-09-21','2026-09-27',NOW(),NOW());
		INSERT INTO study_tasks(id,group_id,week_id,task_type,title,created_at,updated_at)
		VALUES (1,1,1,'weekly_video','Old video',NOW(),NOW()),(2,1,2,'weekly_video','Same video',NOW(),NOW()),
		       (3,1,2,'weekly_book','A',NOW(),NOW()),(4,1,2,'weekly_book','B',NOW(),NOW()),
		       (5,2,3,'weekly_book','Foreign',NOW(),NOW());
		INSERT INTO assets(id,group_id,category,title,original_name,storage_path,created_by,created_at,updated_at)
		VALUES (1,1,'video','Video','video.mp4','video',1,NOW(),NOW());
		INSERT INTO task_assets(group_id,task_id,asset_id,usage_type,sort_order,created_at)
		VALUES (1,1,1,'video',0,NOW()),(1,2,1,'video',0,NOW());
		INSERT INTO checkin_records(group_id,user_id,task_id,week_id,task_type,part,logical_date,checkin_time,created_by,created_at,updated_at)
		VALUES (1,1,1,1,'weekly_video','','2026-09-15',NOW(),1,NOW(),NOW()),
		       (1,1,3,2,'weekly_book','A','2026-09-22',NOW(),1,NOW(),NOW()),
		       (1,1,NULL,NULL,'daily_devotion','','2026-09-22',NOW(),1,NOW(),NOW()),
		       (2,2,5,3,'weekly_book','Foreign','2026-09-22',NOW(),2,NOW(),NOW())`)
	db.SetMaxOpenConns(1)
	repo := &progressQueryRepository{Repository: NewMySQLRepository(db)}
	svc := NewService(repo)
	dates := []string{"2026-09-21", "2026-09-22", "2026-09-27", "2026-09-28"}
	progress, err := svc.GroupProgressForDates(t.Context(), 1, dates)
	if err != nil {
		t.Fatal(err)
	}
	if repo.recordQueries != 2 || repo.taskQueries != 1 {
		t.Fatalf("query count scales with dates: records=%d tasks=%d", repo.recordQueries, repo.taskQueries)
	}
	if progress["2026-09-21"].TasksPerMember != 4 || progress["2026-09-21"].CompletedByUser[1] != 2 {
		t.Fatalf("weekly book and carried video contract changed: %+v", progress["2026-09-21"])
	}
	for _, date := range dates {
		for _, userID := range []uint64{1, 2, 999} {
			today, err := svc.TodayHub(t.Context(), 1, userID, date, nil, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if got := progress[date]; got.TasksPerMember != today.Progress.Total ||
				got.CompletedByUser[userID] != today.Progress.Completed {
				t.Errorf("%s user=%d group=%+v today=%+v", date, userID, got, today.Progress)
			}
		}
	}
	t.Run("empty dates", func(t *testing.T) {
		repo.recordQueries, repo.taskQueries = 0, 0
		got, err := svc.GroupProgressForDates(t.Context(), 1, nil)
		if err != nil || len(got) != 0 || repo.recordQueries != 0 || repo.taskQueries != 0 {
			t.Fatalf("empty input: result=%v err=%v queries=%d/%d", got, err, repo.recordQueries, repo.taskQueries)
		}
	})
}
