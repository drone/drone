package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/session"
	"github.com/gorilla/pat"
)

type ConfigHandler struct {
	conf database.ConfigManager
	sess session.Session
}

func NewConfigHandler(conf database.ConfigManager, sess session.Session) *ConfigHandler {
	return &ConfigHandler{conf, sess}
}

// GetConfig gets the system configuration details.
// GET /api/config
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) error {
	// get the user form the session
	user := h.sess.User(r)
	if user == nil || !user.Admin {
		return notAuthorized{}
	}

	return json.NewEncoder(w).Encode(h.conf.Find())
}

func (h *ConfigHandler) Register(r *pat.Router) {
	r.Get("/v1/config", errorHandler(h.GetConfig))
}
