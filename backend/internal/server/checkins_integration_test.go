//go:build integration

package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agp/backend/internal/audit"
	"agp/backend/internal/checkin"
	"agp/backend/internal/learning"
	"agp/backend/internal/testdb"
	"agp/backend/internal/user"
)

func TestWebAndBotDailyAdmission(t *testing.T) {
	tests := []struct {
		name     string
		daily    string
		taskType string
		want     int
	}{
		{"legacy defaults", `{}`, "daily_devotion", http.StatusCreated},
		{"all disabled", `{"devotion":{"enabled":false},"scripture":{"enabled":false}}`, "daily_devotion", http.StatusBadRequest},
		{"missing custom date", `{"checkin_mode":"separate","devotion":{"enabled":true,"plan_mode":"custom","plans":[{"date":"2026-09-22","title":"Plan"}]},"scripture":{"enabled":false}}`, "daily_devotion", http.StatusBadRequest},
		{"custom date present", `{"checkin_mode":"separate","devotion":{"enabled":true,"plan_mode":"custom","plans":[{"date":"2026-09-23","title":"Plan"}]},"scripture":{"enabled":false}}`, "daily_devotion", http.StatusCreated},
		{"combined scripture remains available", `{"checkin_mode":"combined","devotion":{"enabled":true,"plan_mode":"custom","plans":[]},"scripture":{"enabled":true}}`, "daily_devotion", http.StatusCreated},
		{"separate scripture", `{"checkin_mode":"separate","devotion":{"enabled":false},"scripture":{"enabled":true,"type":"checkin"}}`, "daily_scripture", http.StatusCreated},
		{"combined rejects separate scripture", `{}`, "daily_scripture", http.StatusBadRequest},
	}
	for _, tt := range tests {
		for _, transport := range []string{"web", "bot"} {
			t.Run(tt.name+"/"+transport, func(t *testing.T) {
				db := testdb.Open(t)
				testdb.Exec(t, db, `INSERT INTO study_groups(id,code,name,created_at,updated_at)
					VALUES (1,'a','A',NOW(),NOW());
					INSERT INTO users(id,username,display_name,name_pinyin,created_at,updated_at)
					VALUES (1,'member','Member','member',NOW(),NOW());
					INSERT INTO group_members(group_id,user_id,member_name,joined_at,created_at,updated_at)
					VALUES (1,1,'Member',NOW(),NOW(),NOW())`)
				testdb.Exec(t, db, `INSERT INTO group_settings(group_id,settings,created_at,updated_at)
					VALUES (1,?,NOW(),NOW())`, `{"task_sections":{"daily":`+tt.daily+`}}`)
				checkins := checkin.NewService(checkin.NewMySQLRepository(db))
				a := &app{
					location: time.UTC, db: db,
					users:    user.NewService(user.NewMySQLRepository(db)),
					learning: learning.NewService(learning.NewMySQLRepository(db), nil, checkins),
					checkins: checkins, audits: audit.NewService(audit.NewMySQLRepository(db)),
				}
				for attempt := 0; attempt < 2; attempt++ {
					body := fmt.Sprintf(`{"task_type":%q,"logical_date":"2026-09-23","part":"ignore","task_id":99,"week_id":99}`, tt.taskType)
					if transport == "bot" {
						body = fmt.Sprintf(`{"name":"Member","type":%q,"logicalDate":"2026-09-23"}`, tt.taskType)
					}
					req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
					req.SetPathValue("code", "a")
					req = req.WithContext(context.WithValue(req.Context(), currentUserKey, currentUser{ID: 1, CurrentGroupID: 1}))
					response := httptest.NewRecorder()
					if transport == "bot" {
						a.handleBotCreateCheckin(response, req)
					} else {
						a.handleCreateCheckin(response, req)
					}
					want := tt.want
					if attempt == 1 && want == http.StatusCreated {
						want = http.StatusOK
					}
					if response.Code != want {
						t.Fatalf("status=%d want=%d body=%s", response.Code, want, response.Body)
					}
				}
				var count int
				if err := db.QueryRow(`SELECT COUNT(*) FROM checkin_records`).Scan(&count); err != nil {
					t.Fatal(err)
				}
				wantCount := 0
				if tt.want == http.StatusCreated {
					wantCount = 1
				}
				if count != wantCount {
					t.Fatalf("persisted %d records, want %d", count, wantCount)
				}
				if count == 1 {
					var part string
					var taskID, weekID uint64
					if err := db.QueryRow(`SELECT part,COALESCE(task_id,0),COALESCE(week_id,0) FROM checkin_records`).Scan(&part, &taskID, &weekID); err != nil {
						t.Fatal(err)
					}
					if part != "" || taskID != 0 || weekID != 0 {
						t.Fatalf("daily target not normalized: part=%q task=%d week=%d", part, taskID, weekID)
					}
				}
			})
		}
	}
}
