package http

import (
	"context"
	nethttp "net/http"

	"server/internal/domain/asset"
)

func (a *API) handleRegisterAsset(w nethttp.ResponseWriter, r *nethttp.Request) {
	var src asset.Source
	if !a.read(w, r, &src) {
		return
	}
	res, err := a.service.Register(src)
	if err == nil && a.broadcaster != nil {
		a.broadcaster.BroadcastAsset(res)
	}
	a.reply(w, r, nethttp.StatusCreated, res, err)
}

func (a *API) handleBeginUpload(w nethttp.ResponseWriter, r *nethttp.Request) {
	id := r.PathValue("id")
	res, upload, err := a.service.Begin(r.Context(), id)
	if err == nil && a.broadcaster != nil {
		a.broadcaster.BroadcastAsset(res)
	}
	a.reply(w, r, nethttp.StatusOK, map[string]any{"asset": res, "upload": upload}, err)
}

func (a *API) handleCompleteUpload(w nethttp.ResponseWriter, r *nethttp.Request) {
	a.attempt(a.service.Complete)(w, r)
}

func (a *API) handleFailUpload(w nethttp.ResponseWriter, r *nethttp.Request) {
	a.attempt(func(_ context.Context, id, attempt string) (asset.Asset, error) {
		return a.service.Fail(id, attempt)
	})(w, r)
}

func (a *API) handleAssetContent(w nethttp.ResponseWriter, r *nethttp.Request) {
	id := r.PathValue("id")
	url, err := a.service.Content(r.Context(), id)
	if err != nil {
		a.reply(w, r, 0, nil, err)
		return
	}
	nethttp.Redirect(w, r, url, nethttp.StatusTemporaryRedirect)
}
