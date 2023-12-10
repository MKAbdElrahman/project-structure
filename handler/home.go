package handler

import (
	"counter/component"
	"log/slog"
	"net/http"

	"github.com/alexedwards/scs/v2"
)

var globalCount int

type HomeHandler struct {
	Logger         *slog.Logger
	SessionManager *scs.SessionManager
}

func NewHomeHandler(logger *slog.Logger, sessionManager *scs.SessionManager) *HomeHandler {
	return &HomeHandler{
		SessionManager: sessionManager,
		Logger:         logger,
	}
}
func (h *HomeHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	userCount := h.SessionManager.GetInt(r.Context(), "count")

	data := component.HomeComponetData{
		Global: globalCount,
		User:   userCount,
	}

	h.View(w, r, data)
}

func (h *HomeHandler) HandlePost(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()

	if r.Form.Has("global") {
		globalCount++
	}

	if r.Form.Has("user") {
		currentCount := h.SessionManager.GetInt(r.Context(), "count")
		h.SessionManager.Put(r.Context(), "count", currentCount+1)
	}

	h.HandleGet(w, r)
}

func (h *HomeHandler) View(w http.ResponseWriter, r *http.Request, data component.HomeComponetData) {

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	component.Home(data).Render(r.Context(), w)
}
