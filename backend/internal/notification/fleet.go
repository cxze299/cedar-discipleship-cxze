package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultRobotID   = "default"
	maxRobots        = 32
	maxRobotConfig   = 256 * 1024
	robotHealthy     = "healthy"
	robotDegraded    = "degraded"
	robotUnavailable = "unavailable"
)

var (
	robotIDPattern         = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
	ErrRobotNotFound       = errors.New("robot_not_found")
	ErrRobotAuthentication = errors.New("robot_authentication_failed")
)

type RobotConfig struct {
	ID     string
	Name   string
	Token  string
	Groups map[uint64]Target
}

type RobotStatus struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	State         string        `json:"state"`
	Authenticated bool          `json:"authenticated"`
	Identity      RobotIdentity `json:"identity"`
	LastCheckedAt time.Time     `json:"last_checked_at"`
	ErrorCode     string        `json:"error_code,omitempty"`
	Queue         QueueStats    `json:"queue"`
	Chats         []Chat        `json:"chats"`
	Bindings      []Binding     `json:"bindings"`
}

type fleetRobot struct {
	config  RobotConfig
	client  *PotatoClient
	manager *Manager
}

type Fleet struct {
	robots map[string]*fleetRobot
	order  []string
}

func ParseRobotConfigs(value, legacyToken, legacyGroups string) ([]RobotConfig, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		targets, err := ParseTargets(legacyToken, legacyGroups)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(legacyToken) == "" {
			return nil, nil
		}
		return []RobotConfig{{
			ID: defaultRobotID, Name: "默认机器人", Token: legacyToken, Groups: targets,
		}}, nil
	}
	if len(value) > maxRobotConfig {
		return nil, errors.New("AGP_POTATO_ROBOTS is too large")
	}
	var raw map[string]struct {
		Name   string            `json:"name"`
		Token  string            `json:"token"`
		Groups map[string]Target `json:"groups"`
	}
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil || raw == nil {
		return nil, errors.New("invalid AGP_POTATO_ROBOTS JSON")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, errors.New("invalid AGP_POTATO_ROBOTS JSON")
	}
	if len(raw) == 0 || len(raw) > maxRobots {
		return nil, fmt.Errorf("AGP_POTATO_ROBOTS requires 1-%d robots", maxRobots)
	}

	configs := make([]RobotConfig, 0, len(raw))
	tokens := make(map[string]struct{}, len(raw))
	for id, item := range raw {
		id = strings.TrimSpace(id)
		if !robotIDPattern.MatchString(id) {
			return nil, errors.New("AGP_POTATO_ROBOTS contains an invalid robot ID")
		}
		name := strings.TrimSpace(item.Name)
		if name == "" || len([]rune(name)) > 128 {
			return nil, errors.New("AGP_POTATO_ROBOTS requires robot names of 1-128 characters")
		}
		if !tokenPattern.MatchString(item.Token) {
			return nil, errors.New("AGP_POTATO_ROBOTS contains an invalid token")
		}
		if _, exists := tokens[item.Token]; exists {
			return nil, errors.New("AGP_POTATO_ROBOTS contains a duplicate token")
		}
		tokens[item.Token] = struct{}{}
		targets, err := parseRobotTargets(item.Groups)
		if err != nil {
			return nil, err
		}
		configs = append(configs, RobotConfig{ID: id, Name: name, Token: item.Token, Groups: targets})
	}
	sort.Slice(configs, func(i, j int) bool { return configs[i].ID < configs[j].ID })
	return configs, nil
}

func parseRobotTargets(raw map[string]Target) (map[uint64]Target, error) {
	targets := make(map[uint64]Target, len(raw))
	chatIDs := make(map[int64]struct{}, len(raw))
	for key, target := range raw {
		groupID, err := strconv.ParseUint(key, 10, 64)
		if err != nil || groupID == 0 || strconv.FormatUint(groupID, 10) != key ||
			target.ChatID <= 0 || (target.ChatType != 2 && target.ChatType != 3) {
			return nil, errors.New("AGP_POTATO_ROBOTS groups require positive IDs and chat_type 2 or 3")
		}
		if _, exists := chatIDs[target.ChatID]; exists {
			return nil, errors.New("AGP_POTATO_ROBOTS assigns a robot chat more than once")
		}
		chatIDs[target.ChatID] = struct{}{}
		targets[groupID] = target
	}
	return targets, nil
}

func NewFleet(dir string, configs []RobotConfig, source SnapshotSource) (*Fleet, error) {
	fleet := &Fleet{robots: make(map[string]*fleetRobot, len(configs))}
	for _, config := range configs {
		client, err := NewPotatoClient(config.Token)
		if err != nil {
			return nil, fmt.Errorf("register robot %s: %w", config.ID, err)
		}
		robotDir := filepath.Join(dir, "robots", config.ID)
		if config.ID == defaultRobotID {
			robotDir = dir
		}
		manager, err := NewManager(robotDir, config.Groups, source, client)
		if err != nil {
			return nil, fmt.Errorf("register robot %s: %w", config.ID, err)
		}
		fleet.robots[config.ID] = &fleetRobot{config: config, client: client, manager: manager}
		fleet.order = append(fleet.order, config.ID)
	}
	sort.Strings(fleet.order)
	return fleet, nil
}

func (f *Fleet) Run(ctx context.Context) {
	var workers sync.WaitGroup
	for _, id := range f.order {
		robot := f.robots[id]
		workers.Go(func() { robot.manager.Run(ctx) })
	}
	workers.Wait()
}

func (f *Fleet) Enqueue(event Event) error {
	var errs []error
	for _, id := range f.order {
		if err := f.robots[id].manager.Enqueue(event); err != nil {
			errs = append(errs, fmt.Errorf("robot %s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}

func (f *Fleet) EnqueueInitial(now time.Time) error {
	var errs []error
	for _, id := range f.order {
		if err := f.robots[id].manager.EnqueueInitial(now); err != nil {
			errs = append(errs, fmt.Errorf("robot %s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}

func (f *Fleet) WakeInitial(groupID uint64, now time.Time) error {
	var errs []error
	for _, id := range f.order {
		if err := f.robots[id].manager.WakeInitial(groupID, now); err != nil {
			errs = append(errs, fmt.Errorf("robot %s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}

func (f *Fleet) Robots(ctx context.Context) []RobotStatus {
	statuses := make([]RobotStatus, len(f.order))
	var workers sync.WaitGroup
	for index, id := range f.order {
		robot := f.robots[id]
		workers.Go(func() {
			statuses[index] = robot.status(ctx)
		})
	}
	workers.Wait()
	return statuses
}

func (r *fleetRobot) status(ctx context.Context) RobotStatus {
	status := RobotStatus{
		ID:            r.config.ID,
		Name:          r.config.Name,
		State:         robotHealthy,
		LastCheckedAt: time.Now().UTC(),
		Bindings:      r.manager.Bindings(),
	}
	status.Chats = chatsFromBindings(status.Bindings)
	identity, err := r.client.Identity(ctx)
	if err != nil {
		status.State = robotUnavailable
		status.ErrorCode = "authentication_failed"
	} else {
		status.Authenticated = true
		status.Identity = identity
		chats, err := r.manager.Chats(ctx)
		if err != nil {
			status.State = robotDegraded
			status.ErrorCode = "chat_list_failed"
		} else {
			boundGroups := make(map[int64]uint64, len(status.Bindings))
			for _, binding := range status.Bindings {
				boundGroups[binding.ChatID] = binding.GroupID
			}
			for index := range chats {
				chats[index].GroupID = boundGroups[chats[index].ChatID]
				delete(boundGroups, chats[index].ChatID)
			}
			for _, binding := range status.Bindings {
				if _, missing := boundGroups[binding.ChatID]; missing {
					chats = append(chats, Chat{
						ChatID: binding.ChatID, ChatType: binding.ChatType,
						Title: "已绑定群聊", GroupID: binding.GroupID,
					})
				}
			}
			status.Chats = chats
		}
	}
	stats, err := r.manager.Stats()
	if err != nil {
		if status.State == robotHealthy {
			status.State = robotDegraded
			status.ErrorCode = "queue_status_failed"
		}
	} else {
		status.Queue = stats
	}
	return status
}

func chatsFromBindings(bindings []Binding) []Chat {
	chats := make([]Chat, 0, len(bindings))
	for _, binding := range bindings {
		chats = append(chats, Chat{
			ChatID: binding.ChatID, ChatType: binding.ChatType,
			Title: "已绑定群聊", GroupID: binding.GroupID,
		})
	}
	return chats
}

func (f *Fleet) Assign(
	ctx context.Context,
	robotID string,
	target Target,
	groupID uint64,
	now time.Time,
) error {
	if strings.TrimSpace(robotID) == "" {
		robotID = defaultRobotID
	}
	robot, ok := f.robots[robotID]
	if !ok {
		return ErrRobotNotFound
	}
	if groupID > 0 {
		if _, err := robot.client.Identity(ctx); err != nil {
			return ErrRobotAuthentication
		}
	}
	return robot.manager.Assign(ctx, target, groupID, now)
}

func (f *Fleet) BindingGroupID(robotID string, chatID int64) uint64 {
	if strings.TrimSpace(robotID) == "" {
		robotID = defaultRobotID
	}
	robot, ok := f.robots[robotID]
	if !ok {
		return 0
	}
	for _, binding := range robot.manager.Bindings() {
		if binding.ChatID == chatID {
			return binding.GroupID
		}
	}
	return 0
}
