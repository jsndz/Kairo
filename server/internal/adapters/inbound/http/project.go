package http

import (
	nethttp "net/http"
)

func (a *API) handleHealth(w nethttp.ResponseWriter, r *nethttp.Request) {
	a.reply(w, r, nethttp.StatusOK, map[string]string{"status": "ok"}, nil)
}
