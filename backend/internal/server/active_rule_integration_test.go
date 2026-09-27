//go:build integration

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agp/backend/internal/audit"
	"agp/backend/internal/learning"
	"agp/backend/internal/testdb"
)

func TestActiveRulePreservesConcurrentLearningConfig(t *testing.T) {
	db := testdb.Open(t)
	testdb.Exec(t, db, `INSERT INTO group_settings(group_id,settings,created_at,updated_at)
		VALUES (1,'{"task_sections":{"daily":{"devotion":{"title":"old"}}}}',NOW(),NOW())`)
	a := &app{
		learning: learning.NewService(learning.NewMySQLRepository(db)),
		audits:   audit.NewService(audit.NewMySQLRepository(db)),
	}
	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(t.Context(), `UPDATE group_settings
		SET settings='{"task_sections":{"daily":{"devotion":{"title":"new"}}},"other_setting":true}' WHERE group_id=1`); err != nil {
		t.Fatal(err)
	}
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		ctx = context.WithValue(ctx, currentUserKey, currentUser{ID: 1, CurrentGroupID: 1})
		req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"mode":"all","task_types":["daily_devotion"]}`)).WithContext(ctx)
		response := httptest.NewRecorder()
		a.handleDashboardActiveRule(response, req)
		done <- response
	}()
	testdb.WaitForLockWait(t, db)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	response := <-done
	if response.Code != http.StatusOK {
		t.Fatalf("update active rule: %d %s", response.Code, response.Body)
	}
	var title, mode, other string
	if err := db.QueryRow(`SELECT settings->>'$.task_sections.daily.devotion.title',
		settings->>'$.active_member_rule.mode',COALESCE(settings->>'$.other_setting','missing')
		FROM group_settings WHERE group_id=1`).Scan(&title, &mode, &other); err != nil {
		t.Fatal(err)
	}
	if title != "new" || mode != "all" || other != "true" {
		t.Fatalf("settings overwritten: title=%s mode=%s other=%s", title, mode, other)
	}
	for _, groupID := range []uint64{2, 3} {
		if groupID == 3 {
			testdb.Exec(t, db, `INSERT INTO group_settings(group_id,settings,created_at,updated_at) VALUES (3,'null',NOW(),NOW())`)
		}
		if err := a.learning.SaveActiveMemberRule(t.Context(), groupID, map[string]any{"mode": "any", "task_types": []string{"daily_devotion"}}); err != nil {
			t.Fatal(err)
		}
		settings, err := a.learning.LearningConfig(t.Context(), groupID)
		if err != nil {
			t.Fatal(err)
		}
		if rule := activeMemberRuleFromSettings(settings); rule.Mode != "any" || len(rule.TaskTypes) != 1 {
			t.Fatalf("initial settings group=%d rule=%+v", groupID, rule)
		}
	}
}
