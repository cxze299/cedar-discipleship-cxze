package server

import (
	"errors"
	"net/http"
	"time"

	notificationdomain "agp/backend/internal/notification"
)

func (a *app) handleBotManagement(w http.ResponseWriter, r *http.Request) {
	user := mustUser(r)
	if a.botManager == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"configured":   false,
			"robots":       []any{},
			"study_groups": user.Groups,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"configured":   true,
		"robots":       a.botManager.Robots(r.Context()),
		"study_groups": user.Groups,
	})
}

func (a *app) handleBotBinding(w http.ResponseWriter, r *http.Request) {
	if a.botManager == nil {
		writeError(w, http.StatusServiceUnavailable, "bot_not_configured")
		return
	}
	var req struct {
		RobotID  string `json:"robot_id"`
		ChatID   int64  `json:"chat_id"`
		ChatType int    `json:"chat_type"`
		GroupID  uint64 `json:"group_id"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.RobotID == "" {
		req.RobotID = "default"
	}
	if req.ChatID <= 0 || (req.ChatType != 2 && req.ChatType != 3) {
		writeError(w, http.StatusBadRequest, "invalid_bot_chat")
		return
	}
	user := mustUser(r)
	if req.GroupID > 0 {
		found := false
		for _, group := range user.Groups {
			if group.ID == req.GroupID {
				found = true
				break
			}
		}
		if !found {
			writeError(w, http.StatusBadRequest, "study_group_not_found")
			return
		}
	}
	previousGroupID := a.botManager.BindingGroupID(req.RobotID, req.ChatID)
	target := notificationdomain.Target{ChatID: req.ChatID, ChatType: req.ChatType}
	if err := a.botManager.Assign(r.Context(), req.RobotID, target, req.GroupID, time.Now().UTC()); err != nil {
		if errors.Is(err, notificationdomain.ErrRobotNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, notificationdomain.ErrChatNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, notificationdomain.ErrRobotAuthentication) {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "bot_binding_save_failed")
		return
	}
	a.audit(0, user.ID, "save_bot_binding", "potato_chat", uint64(req.ChatID),
		map[string]any{"robot_id": req.RobotID, "group_id": previousGroupID},
		map[string]any{"robot_id": req.RobotID, "group_id": req.GroupID, "chat_type": req.ChatType},
		r,
	)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
