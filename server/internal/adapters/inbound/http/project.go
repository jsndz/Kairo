package http

import (
	nethttp "net/http"

	"server/internal/domain/timeline"
)

func (a *API) handleHealth(w nethttp.ResponseWriter, r *nethttp.Request) {
	a.reply(w, r, nethttp.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (a *API) handleGetProject(w nethttp.ResponseWriter, r *nethttp.Request) {
	a.reply(w, r, nethttp.StatusOK, a.service.Get(), nil)
}

func (a *API) handleCommitOperation(w nethttp.ResponseWriter, r *nethttp.Request) {
	var edit timeline.Edit
	if !a.read(w, r, &edit) {
		return
	}
	state, err := a.service.Commit(edit)
	if err == nil && a.broadcaster != nil {
		a.broadcaster.BroadcastOperation(state.ID, edit, state, "")
	}
	a.reply(w, r, nethttp.StatusOK, state, err)
}
