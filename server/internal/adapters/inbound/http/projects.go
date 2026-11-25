package http

import (
	"context"
	nethttp "net/http"

	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
	domainExport "server/internal/domain/export"
	"server/internal/domain/timeline"
	"server/internal/ports"
)

type createProjectRequest struct {
	Name string `json:"name"`
}

type createRoomResponse struct {
	RoomID string `json:"roomId"`
	Name   string `json:"name"`
}

func (a *API) handleCreateRoom(w nethttp.ResponseWriter, r *nethttp.Request) {
	if a.manager == nil {
		a.reply(w, r, 0, nil, domErrors.ErrInvalid)
		return
	}

	var req createProjectRequest
	_ = a.read(w, r, &req)
	name := req.Name
	if name == "" {
		name = "Collaboration Room"
	}

	summary, _, err := a.manager.Create(r.Context(), name)
	if err != nil {
		a.reply(w, r, 0, nil, err)
		return
	}

	a.reply(w, r, nethttp.StatusCreated, createRoomResponse{
		RoomID: summary.ID,
		Name:   summary.Name,
	}, nil)
}

func (a *API) handleCreateProject(w nethttp.ResponseWriter, r *nethttp.Request) {
	if a.manager == nil {
		a.reply(w, r, 0, nil, domErrors.ErrInvalid)
		return
	}

	var req createProjectRequest
	_ = a.read(w, r, &req)

	summary, _, err := a.manager.Create(r.Context(), req.Name)
	a.reply(w, r, nethttp.StatusCreated, summary, err)
}

func (a *API) handleGetProjectByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	projID := r.PathValue("projectId")
	if a.manager == nil || projID == "" {
		a.reply(w, r, 0, nil, domErrors.ErrNotFound)
		return
	}

	inst, err := a.manager.Get(r.Context(), projID)
	if err != nil {
		a.reply(w, r, 0, nil, domErrors.ErrNotFound)
		return
	}
	a.reply(w, r, nethttp.StatusOK, inst.Get(), nil)
}

func (a *API) handleDeleteProjectByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	projID := r.PathValue("projectId")
	if a.manager == nil || projID == "" {
		a.reply(w, r, 0, nil, domErrors.ErrNotFound)
		return
	}

	err := a.manager.Delete(r.Context(), projID)
	a.reply(w, r, nethttp.StatusOK, map[string]string{"status": "deleted"}, err)
}

func (a *API) handleCommitOperationByProject(w nethttp.ResponseWriter, r *nethttp.Request) {
	projID := r.PathValue("projectId")
	var edit timeline.Edit
	if !a.read(w, r, &edit) {
		return
	}

	svc, err := a.resolveService(r, projID)
	if err != nil {
		a.reply(w, r, 0, nil, domErrors.ErrNotFound)
		return
	}

	state, err := svc.Commit(edit)
	if err == nil && a.broadcaster != nil {
		a.broadcaster.BroadcastOperation(state.ID, edit, state, "")
	}
	a.reply(w, r, nethttp.StatusOK, state, err)
}

func (a *API) handleRegisterAssetByProject(w nethttp.ResponseWriter, r *nethttp.Request) {
	projID := r.PathValue("projectId")
	var src asset.Source
	if !a.read(w, r, &src) {
		return
	}

	svc, err := a.resolveService(r, projID)
	if err != nil {
		a.reply(w, r, 0, nil, domErrors.ErrNotFound)
		return
	}

	res, err := svc.Register(src)
	if err == nil && a.broadcaster != nil {
		a.broadcaster.BroadcastProjectAsset(projID, res)
	}
	a.reply(w, r, nethttp.StatusCreated, res, err)
}

func (a *API) handleBeginUploadByProject(w nethttp.ResponseWriter, r *nethttp.Request) {
	projID := r.PathValue("projectId")
	assetID := r.PathValue("id")

	svc, err := a.resolveService(r, projID)
	if err != nil {
		a.reply(w, r, 0, nil, domErrors.ErrNotFound)
		return
	}

	res, upload, err := svc.Begin(r.Context(), assetID)
	if err == nil && a.broadcaster != nil {
		a.broadcaster.BroadcastProjectAsset(projID, res)
	}
	a.reply(w, r, nethttp.StatusOK, map[string]any{"asset": res, "upload": upload}, err)
}

func (a *API) handleCompleteUploadByProject(w nethttp.ResponseWriter, r *nethttp.Request) {
	projID := r.PathValue("projectId")
	svc, err := a.resolveService(r, projID)
	if err != nil {
		a.reply(w, r, 0, nil, domErrors.ErrNotFound)
		return
	}

	a.attempt(func(ctx context.Context, id, attempt string) (asset.Asset, error) {
		res, err := svc.Complete(ctx, id, attempt)
		if err == nil && a.broadcaster != nil {
			a.broadcaster.BroadcastProjectAsset(projID, res)
		}
		return res, err
	})(w, r)
}

func (a *API) handleFailUploadByProject(w nethttp.ResponseWriter, r *nethttp.Request) {
	projID := r.PathValue("projectId")
	svc, err := a.resolveService(r, projID)
	if err != nil {
		a.reply(w, r, 0, nil, domErrors.ErrNotFound)
		return
	}

	a.attempt(func(_ context.Context, id, attempt string) (asset.Asset, error) {
		res, err := svc.Fail(id, attempt)
		if err == nil && a.broadcaster != nil {
			a.broadcaster.BroadcastProjectAsset(projID, res)
		}
		return res, err
	})(w, r)
}

func (a *API) handleAssetContentByProject(w nethttp.ResponseWriter, r *nethttp.Request) {
	projID := r.PathValue("projectId")
	assetID := r.PathValue("id")

	svc, err := a.resolveService(r, projID)
	if err != nil {
		a.reply(w, r, 0, nil, domErrors.ErrNotFound)
		return
	}

	url, err := svc.Content(r.Context(), assetID)
	if err != nil {
		a.reply(w, r, 0, nil, err)
		return
	}
	nethttp.Redirect(w, r, url, nethttp.StatusTemporaryRedirect)
}

func (a *API) handleCreateExportByProject(w nethttp.ResponseWriter, r *nethttp.Request) {
	projID := r.PathValue("projectId")
	var body createExportRequest
	if !a.read(w, r, &body) {
		return
	}

	if projID == "" && body.ProjectID != "" {
		projID = body.ProjectID
	}

	settings := domainExport.DefaultSettings()
	if body.Settings != nil {
		settings = *body.Settings
		if settings.Method == "" {
			settings.Method = domainExport.MethodServer
		}
	} else {
		if body.Format != "" {
			settings.Format = body.Format
		}
		if body.Filename != "" {
			settings.Filename = body.Filename
		}
		if body.Resolution != "" {
			settings.Resolution = body.Resolution
		}
		if body.FrameRate != 0 {
			settings.FrameRate = body.FrameRate
		}
		if body.Quality != "" {
			settings.Quality = body.Quality
		}
		if body.Method != "" {
			settings.Method = body.Method
		}
	}

	req := ports.SubmitRequest{
		ProjectID: projID,
		RequestID: body.RequestID,
		Revision:  body.Revision,
		Settings:  settings,
	}

	job, err := a.exportSvc.Submit(r.Context(), req)
	a.reply(w, r, nethttp.StatusOK, job, err)
}

func (a *API) resolveService(r *nethttp.Request, projID string) (ports.ProjectInstance, error) {
	if a.manager == nil || projID == "" {
		return nil, domErrors.ErrNotFound
	}
	return a.manager.Get(r.Context(), projID)
}
