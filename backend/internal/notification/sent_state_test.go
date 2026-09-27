package notification

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSentStateStoreBootstrapsCompletedNotifications(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	completed := filepath.Join(dir, "completed")
	if err := os.MkdirAll(completed, 0o700); err != nil {
		t.Fatal(err)
	}
	target := Target{ChatID: 99, ChatType: 3}
	expiresAt := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	item := job{
		Event: Event{
			RecordID:    10,
			GroupID:     1,
			LogicalDate: "2026-09-10",
			OccurredAt:  time.Date(2026, 9, 10, 8, 0, 0, 0, time.UTC),
		},
		Target:    target,
		Messages:  []string{"每日灵修\n1 【新】张三"},
		ExpiresAt: expiresAt,
		NextPart:  1,
		Status:    "sent",
	}
	if err := writeJob(filepath.Join(completed, "legacy.json"), item); err != nil {
		t.Fatal(err)
	}
	store, err := newSentStateStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	content := canonicalNotificationContent("每日灵修\n1 张三")
	needsSend, err := store.NeedsSend(sentState{
		GroupID: 1, Target: target, Topic: "daily", Version: "daily:2026-09-10", Hash: contentHash(content),
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if needsSend {
		t.Fatal("completed notification was not migrated")
	}
	// Old last-sent filenames have no reliable group identity. Only completed
	// jobs may reconstruct that association during an upgrade.
	if err := os.WriteFile(filepath.Join(store.dir, "00000000000000000099-3-daily.json"),
		[]byte(`{"target":{"chat_id":99,"chat_type":3},"topic":"daily","version":"daily:2026-09-30","hash":"old"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	restarted, err := newSentStateStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, groupID := range []uint64{1, 2} {
		needsSend, err := restarted.NeedsSend(sentState{
			GroupID: groupID, Target: target, Topic: "daily", Version: "daily:2026-09-10", Hash: contentHash(content),
		}, 0)
		if err != nil || needsSend != (groupID == 2) {
			t.Fatalf("group %d after legacy upgrade: send=%v err=%v", groupID, needsSend, err)
		}
	}
}
