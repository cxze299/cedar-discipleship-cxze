package notification

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestParseRobotConfigs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		value        string
		legacyToken  string
		legacyGroups string
		want         []RobotConfig
		wantErr      bool
	}{
		{
			name:         "disabled",
			legacyGroups: "",
		},
		{
			name:         "legacy default",
			legacyToken:  "123:secret",
			legacyGroups: `{"1":{"chat_id":99,"chat_type":3}}`,
			want: []RobotConfig{{
				ID: "default", Name: "默认机器人", Token: "123:secret",
				Groups: map[uint64]Target{1: {ChatID: 99, ChatType: 3}},
			}},
		},
		{
			name: "multiple robots sorted",
			value: `{
				"secondary":{"name":"备用机器人","token":"456:secret","groups":{"2":{"chat_id":99,"chat_type":2}}},
				"primary":{"name":"主机器人","token":"123:secret","groups":{"1":{"chat_id":99,"chat_type":3}}}
			}`,
			want: []RobotConfig{
				{
					ID: "primary", Name: "主机器人", Token: "123:secret",
					Groups: map[uint64]Target{1: {ChatID: 99, ChatType: 3}},
				},
				{
					ID: "secondary", Name: "备用机器人", Token: "456:secret",
					Groups: map[uint64]Target{2: {ChatID: 99, ChatType: 2}},
				},
			},
		},
		{
			name:    "invalid id",
			value:   `{"Bad ID":{"name":"bad","token":"123:secret"}}`,
			wantErr: true,
		},
		{
			name:    "duplicate token",
			value:   `{"one":{"name":"one","token":"123:secret"},"two":{"name":"two","token":"123:secret"}}`,
			wantErr: true,
		},
		{
			name:    "unknown field",
			value:   `{"one":{"name":"one","token":"123:secret","enabled":true}}`,
			wantErr: true,
		},
		{
			name:    "duplicate chat in one robot",
			value:   `{"one":{"name":"one","token":"123:secret","groups":{"1":{"chat_id":99,"chat_type":2},"2":{"chat_id":99,"chat_type":2}}}}`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRobotConfigs(tt.value, tt.legacyToken, tt.legacyGroups)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseRobotConfigs() error = %v, wantErr=%v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseRobotConfigs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestFleetUsesIndependentRobotDirectories(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fleet, err := NewFleet(dir, []RobotConfig{
		{ID: "default", Name: "default", Token: "123:secret"},
		{ID: "secondary", Name: "secondary", Token: "456:secret"},
	}, &fakeSource{})
	if err != nil {
		t.Fatal(err)
	}
	if fleet.robots["default"].manager.queue.dir != dir {
		t.Fatalf("default dir = %q, want %q", fleet.robots["default"].manager.queue.dir, dir)
	}
	if fleet.robots["secondary"].manager.queue.dir == dir {
		t.Fatal("secondary robot reused the legacy queue directory")
	}
}

func TestFleetScopesStatusAndBindingsByRobot(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/getMe":
			_, _ = io.WriteString(w, `{"ok":true,"result":{"id":101,"first_name":"Bot","username":"test_bot"}}`)
		case "/getGroups":
			_, _ = io.WriteString(w, `{"ok":true,"result":{"Groups":[],"SuperGroups":[{"PeerID":20,"PeerName":"Study"}]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	fleet, err := NewFleet(t.TempDir(), []RobotConfig{
		{ID: "primary", Name: "primary", Token: "123:secret"},
		{ID: "secondary", Name: "secondary", Token: "456:secret"},
	}, &fakeSource{snapshot: Snapshot{Text: "current", ExpiresAt: time.Now().Add(time.Hour)}})
	if err != nil {
		t.Fatal(err)
	}
	for _, robot := range fleet.robots {
		robot.client.identityEndpoint = server.URL + "/getMe"
		robot.client.groupsEndpoint = server.URL + "/getGroups"
	}
	target := Target{ChatID: 20, ChatType: 3}
	if err := fleet.Assign(t.Context(), "primary", target, 1, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := fleet.Assign(t.Context(), "secondary", target, 2, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := fleet.BindingGroupID("primary", 20); got != 1 {
		t.Fatalf("primary group = %d, want 1", got)
	}
	if got := fleet.BindingGroupID("secondary", 20); got != 2 {
		t.Fatalf("secondary group = %d, want 2", got)
	}

	statuses := fleet.Robots(t.Context())
	if len(statuses) != 2 {
		t.Fatalf("statuses = %#v", statuses)
	}
	for _, status := range statuses {
		if !status.Authenticated || status.State != robotHealthy || len(status.Chats) != 1 {
			t.Fatalf("status = %#v", status)
		}
		if status.Chats[0].GroupID != fleet.BindingGroupID(status.ID, 20) {
			t.Fatalf("chat binding = %#v", status.Chats[0])
		}
	}
}
