package http

import (
	nethttp "net/http"

	domainExport "server/internal/domain/export"
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
