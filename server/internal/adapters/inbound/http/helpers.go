package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	nethttp "net/http"

	"server/internal/domain/asset"
	domErrors "server/internal/domain/errors"
)

const maxRequestBodyBytes = 256 << 10

func (a *API) attempt(apply func(context.Context, string, string) (asset.Asset, error)) nethttp.HandlerFunc {
	return func(w nethttp.ResponseWriter, r *nethttp.Request) {
		var body struct {
			Attempt string `json:"attempt"`
		}
		if !a.read(w, r, &body) {
			return
		}
		res, err := apply(r.Context(), r.PathValue("id"), body.Attempt)
		a.reply(w, r, nethttp.StatusOK, res, err)
	}
}

func (a *API) read(w nethttp.ResponseWriter, r *nethttp.Request, dst any) bool {
	contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if contentType != "application/json" {
		a.reply(w, r, 0, nil, domErrors.ErrInvalid)
		return false
	}

	decoder := json.NewDecoder(nethttp.MaxBytesReader(w, r.Body, maxRequestBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil || decoder.Decode(new(any)) != io.EOF {
		a.reply(w, r, 0, nil, domErrors.ErrInvalid)
		return false
	}
	return true
}

func (a *API) reply(w nethttp.ResponseWriter, r *nethttp.Request, status int, data any, err error) {
	if err != nil {
		switch {
		case errors.Is(err, domErrors.ErrNotFound):
			nethttp.Error(w, `{"error":"not found"}`, nethttp.StatusNotFound)
		case errors.Is(err, domErrors.ErrConflict):
			nethttp.Error(w, `{"error":"conflict"}`, nethttp.StatusConflict)
		case errors.Is(err, domErrors.ErrUnavailable):
			nethttp.Error(w, `{"error":"storage unavailable"}`, nethttp.StatusServiceUnavailable)
		default:
			nethttp.Error(w, `{"error":"invalid request"}`, nethttp.StatusUnprocessableEntity)
		}
		return
	}

	if status == 0 {
		status = nethttp.StatusOK
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
