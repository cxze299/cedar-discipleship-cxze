//go:build integration

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"agp/backend/internal/learning"
	"agp/backend/internal/testdb"
)

func migrationOptions(t *testing.T, db *sql.DB, config string) options {
	t.Helper()
	var name string
	if err := db.QueryRow("SELECT DATABASE()").Scan(&name); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	return options{
		dsn:       fmt.Sprintf("root@tcp(%s)/%s?parseTime=true", os.Getenv("CEDAR_TEST_MYSQL_ADDR"), name),
		groupCode: "review", groupName: "Review", defaultPassword: "test-password",
		configPath: configPath, skipRecords: true, reportDir: filepath.Join(dir, "reports"),
	}
}

func TestMigrationFailuresDoNotPersistPartialResults(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		for _, failMembers := range []bool{true, false} {
			t.Run(fmt.Sprintf("dry=%v/fail=%v", dryRun, failMembers), func(t *testing.T) {
				db := testdb.Open(t)
				opt := migrationOptions(t, db, `{"members":["未映射测试成员"],"task_sections":{}}`)
				opt.dryRun, opt.failOnGeneratedUsernames = dryRun, failMembers
				err := run(opt)
				if (err != nil) != failMembers {
					t.Errorf("run error=%v, want failure=%v", err, failMembers)
				}
				for _, table := range []string{"study_groups", "group_settings", "users", "group_members"} {
					var count int
					if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
						t.Fatal(err)
					}
					want := 0
					if !dryRun && !failMembers {
						want = 1
					}
					if count != want {
						t.Errorf("%s persisted rows=%d, want %d", table, count, want)
					}
				}
				files, err := filepath.Glob(filepath.Join(opt.reportDir, "*.json"))
				if err != nil || len(files) != 1 {
					t.Fatalf("report missing: %v %v", files, err)
				}
				data, err := os.ReadFile(files[0])
				if err != nil {
					t.Fatal(err)
				}
				var report migrationReport
				if err := json.Unmarshal(data, &report); err != nil {
					t.Fatal(err)
				}
				if (len(report.Failures) > 0) != failMembers {
					t.Fatalf("failure evidence missing: %+v", report)
				}
				wantOutcome := "committed"
				if dryRun {
					wantOutcome = "planned"
				}
				if failMembers {
					wantOutcome = "rolled_back"
					if dryRun {
						wantOutcome = "blocked"
					}
				}
				if report.Outcome != wantOutcome {
					t.Fatalf("outcome=%q, want %q", report.Outcome, wantOutcome)
				}
			})
		}
	}
}

func TestForceMigrationPreservesReferencedVideoHistory(t *testing.T) {
	db := testdb.Open(t)
	testdb.Exec(t, db, `INSERT INTO study_groups(id,code,name,created_at,updated_at)
		VALUES(1,'review','Review',NOW(),NOW());
		INSERT INTO study_weeks(id,group_id,start_date,end_date,created_at,updated_at)
		VALUES(1,1,'2026-09-14','2026-09-20',NOW(),NOW()),(2,1,'2026-09-21','2026-09-27',NOW(),NOW());
		INSERT INTO study_tasks(id,group_id,week_id,task_type,title,created_at,updated_at)
		VALUES(1,1,1,'weekly_video','Old video',NOW(),NOW()),(2,1,1,'weekly_book','Unused',NOW(),NOW()),
		      (3,1,2,'weekly_video','Same video',NOW(),NOW());
		INSERT INTO assets(id,group_id,category,title,original_name,storage_path,created_by,created_at,updated_at)
		VALUES(1,1,'video','Video','video.mp4','video',1,NOW(),NOW());
		INSERT INTO task_assets(group_id,task_id,asset_id,usage_type,created_at)
		VALUES(1,1,1,'video',NOW()),(1,2,1,'reading',NOW()),(1,3,1,'video',NOW());
		INSERT INTO checkin_records(group_id,user_id,task_id,week_id,task_type,logical_date,checkin_time,created_by,created_at,updated_at)
		VALUES(1,1,1,1,'weekly_video','2026-09-15',NOW(),1,NOW(),NOW())`)
	opt := migrationOptions(t, db, `{"task_sections":{},"weekly_schedule":[
		{"start":"2026-09-14","end":"2026-09-20","title":"Replacement","video":"New video"}]}`)
	opt.forceOverwrite = true
	if err := run(opt); err != nil {
		t.Fatal(err)
	}
	var weekID sql.NullInt64
	var enabled int
	if err := db.QueryRow("SELECT week_id,enabled FROM study_tasks WHERE id=1").Scan(&weekID, &enabled); err != nil {
		t.Fatalf("referenced task lost: %v", err)
	}
	if weekID.Valid || enabled != 0 {
		t.Fatalf("old task not retired: week=%v enabled=%d", weekID, enabled)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM task_assets WHERE task_id=2").Scan(&count); err != nil || count != 0 {
		t.Fatalf("unused binding remains: %d %v", count, err)
	}
	records, err := learning.NewMySQLRepository(db).ListCompletionRecords(t.Context(), 1, 1, "2026-09-21", "2026-09-27")
	if err != nil || len(records) != 1 || records[0].AssetID != 1 {
		t.Fatalf("cross-week video completion lost: %+v %v", records, err)
	}
}
