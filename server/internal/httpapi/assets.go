package httpapi

import (
	"context"
	"net/http"

	"server/internal/domain/asset"
)

func (a *API) handleRegisterAsset(w http.ResponseWriter, r *http.Request) {
	var src asset.Source
	if !a.read(w, r, &src) {
		return
	}
	res, err := a.service.Register(src)
	if err == nil && a.hub != nil {
		a.hub.BroadcastAsset(res)
	}
	a.reply(w, r, http.StatusCreated, res, err)
}

func (a *API) handleBeginUpload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, upload, err := a.service.Begin(r.Context(), id)
	if err == nil && a.hub != nil {
		a.hub.BroadcastAsset(res)
	}
	a.reply(w, r, http.StatusOK, map[string]any{"asset": res, "upload": upload}, err)
}

func (a *API) handleCompleteUpload(w http.ResponseWriter, r *http.Request) {
	a.attempt(a.service.Complete)(w, r)
}

func (a *API) handleFailUpload(w http.ResponseWriter, r *http.Request) {
	a.attempt(func(_ context.Context, id, attempt string) (asset.Asset, error) {
		return a.service.Fail(id, attempt)
	})(w, r)
}

func (a *API) handleAssetContent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	url, err := a.service.Content(r.Context(), id)
	if err != nil {
		a.reply(w, r, 0, nil, err)
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
