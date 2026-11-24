package http

import (
	nethttp "net/http"

	domainExport "server/internal/domain/export"
	"server/internal/ports"
)

type createExportRequest struct {
	RequestID  string                  `json:"requestId"`
	Revision   uint64                  `json:"revision"`
	Settings   *domainExport.Settings  `json:"settings,omitempty"`
	Format     domainExport.Format     `json:"format,omitempty"`
	Filename   string                  `json:"filename,omitempty"`
	Resolution domainExport.Resolution `json:"resolution,omitempty"`
	FrameRate  domainExport.FrameRate  `json:"frameRate,omitempty"`
	Quality    domainExport.Quality    `json:"quality,omitempty"`
	Method     domainExport.Method     `json:"method,omitempty"`
}

func (a *API) handleCreateExport(w nethttp.ResponseWriter, r *nethttp.Request) {
	var body createExportRequest
	if !a.read(w, r, &body) {
		return
	}

	settings := domainExport.DefaultSettings()
	if body.Settings != nil {
		settings = *body.Settings
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
		RequestID: body.RequestID,
		Revision:  body.Revision,
		Settings:  settings,
	}

	job, err := a.exportSvc.Submit(r.Context(), req)
	a.reply(w, r, nethttp.StatusOK, job, err)
}

func (a *API) handleGetExport(w nethttp.ResponseWriter, r *nethttp.Request) {
	id := r.PathValue("id")
	job, err := a.exportSvc.Get(r.Context(), id)
	a.reply(w, r, nethttp.StatusOK, job, err)
}

func (a *API) handleCancelExport(w nethttp.ResponseWriter, r *nethttp.Request) {
	id := r.PathValue("id")
	job, err := a.exportSvc.Cancel(r.Context(), id)
	a.reply(w, r, nethttp.StatusOK, job, err)
}

func (a *API) handleRetryExport(w nethttp.ResponseWriter, r *nethttp.Request) {
	id := r.PathValue("id")
	job, err := a.exportSvc.Retry(r.Context(), id)
	a.reply(w, r, nethttp.StatusOK, job, err)
}

func (a *API) handleDownloadExport(w nethttp.ResponseWriter, r *nethttp.Request) {
	id := r.PathValue("id")
	downloadURL, err := a.exportSvc.DownloadURL(r.Context(), id)
	if err != nil {
		a.reply(w, r, 0, nil, err)
		return
	}
	nethttp.Redirect(w, r, downloadURL, nethttp.StatusTemporaryRedirect)
}
