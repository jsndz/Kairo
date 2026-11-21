package httpapi

import (
	"net/http"

	"server/internal/domain/timeline"
)

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	a.reply(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (a *API) handleGetProject(w http.ResponseWriter, r *http.Request) {
	a.reply(w, r, http.StatusOK, a.service.Get(), nil)
}

func (a *API) handleCommitOperation(w http.ResponseWriter, r *http.Request) {
	var edit timeline.Edit
	if !a.read(w, r, &edit) {
		return
	}
	state, err := a.service.Commit(edit)
	a.reply(w, r, http.StatusOK, state, err)
}
