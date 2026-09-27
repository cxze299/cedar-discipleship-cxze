package notification

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

type orderedSource struct {
	canceled bool
}

func (*orderedSource) Enabled(context.Context, Event) (bool, error) { return true, nil }

func (s *orderedSource) Snapshot(_ context.Context, event Event) (Snapshot, error) {
	if event.Initial == "weekly" {
		return Snapshot{}, nil
	}
	text := "每日灵修\n1 张三\n2 李四"
	if event.RecordID == 10 || s.canceled {
		text = "每日灵修\n1 张三"
	}
	snapshot := Snapshot{CoveredRecordID: 20}
	snapshot.Text = text
	snapshot.Topic = "daily"
	snapshot.Version = "daily:2026-09-27"
	snapshot.ExpiresAt = time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	return snapshot, nil
}

func TestQueueFullSummaryCoversOldEventsAcrossRestart(t *testing.T) {
	source, sender := &orderedSource{}, &fakeSender{}
	targets := map[uint64][]Target{1: {{ChatID: 99, ChatType: 3}}}
	dir := t.TempDir()
	now := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	queue, err := NewQueue(dir, targets, source, sender)
	if err != nil {
		t.Fatal(err)
	}
	if err := queue.EnqueueInitial(now); err != nil {
		t.Fatal(err)
	}
	queue.processNext(t.Context(), now)
	queue, err = NewQueue(dir, targets, source, sender)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint64{10, 20} {
		if err := queue.Enqueue(Event{RecordID: id, GroupID: 1, LogicalDate: "2026-09-27", OccurredAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	for range 3 {
		queue.processNext(t.Context(), now)
	}
	want := []string{"每日灵修\n1 张三\n2 李四"}
	if !reflect.DeepEqual(sender.messages, want) {
		t.Fatalf("old events regressed full progress: %#v", sender.messages)
	}
	source.canceled = true
	if err := queue.WakeInitial(1, now); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		queue.processNext(t.Context(), now)
	}
	want = append(want, "每日灵修\n1 张三")
	if !reflect.DeepEqual(sender.messages, want) {
		t.Fatalf("real cancellation was suppressed: %#v", sender.messages)
	}
}

func TestQueueRebindingDoesNotReuseAnotherGroupsSentState(t *testing.T) {
	now := time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)
	target := Target{ChatID: 99, ChatType: 3}
	source := &fakeSource{snapshot: Snapshot{
		Text: "本周任务\n1 A", Topic: "weekly", Version: "weekly:2026-09-27", ExpiresAt: now.Add(48 * time.Hour),
	}}
	sender := &fakeSender{}
	dir := t.TempDir()
	queue, err := NewQueue(dir, map[uint64][]Target{1: {target}}, source, sender)
	if err != nil {
		t.Fatal(err)
	}
	if err := queue.Enqueue(Event{RecordID: 1, GroupID: 1, LogicalDate: "2026-09-26", OccurredAt: now}); err != nil {
		t.Fatal(err)
	}
	queue.processNext(t.Context(), now)
	source.snapshot.Text = "本周任务\n1 B"
	source.snapshot.Version = "weekly:2026-09-26"
	queue, err = NewQueue(dir, map[uint64][]Target{2: {target}}, source, sender)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint64{2, 3} {
		if err := queue.Enqueue(Event{RecordID: id, GroupID: 2, LogicalDate: "2026-09-26", OccurredAt: now}); err != nil {
			t.Fatal(err)
		}
		queue.processNext(t.Context(), now)
		queue, err = NewQueue(dir, map[uint64][]Target{2: {target}}, source, sender)
		if err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(sender.messages, []string{"本周任务\n1 A", "本周任务\n1 B"}) {
		t.Fatalf("rebind/restart messages = %#v", sender.messages)
	}
}

func TestQueueFullSummarySupersedesFrozenEventRemainder(t *testing.T) {
	queue, source, sender, event, now := queueFixture(t)
	source.snapshot.Text = "每日灵修\n" + strings.Repeat("旧进度\n", 700)
	chunks := splitMessage(source.snapshot.Text)
	if len(chunks) < 2 {
		t.Fatal("fixture requires multiple message parts")
	}
	if err := queue.Enqueue(event); err != nil {
		t.Fatal(err)
	}
	queue.processNext(t.Context(), now)
	source.snapshot.Text = "每日灵修\n当前完整进度"
	source.snapshot.CoveredRecordID = event.RecordID
	if err := queue.EnqueueInitial(now); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		queue.processNext(t.Context(), now)
	}
	if !reflect.DeepEqual(sender.messages, []string{chunks[0], source.snapshot.Text}) {
		t.Fatalf("frozen old remainder sent after full summary: %d parts", len(sender.messages))
	}
}
