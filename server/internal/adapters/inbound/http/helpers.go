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
	if err := decoder.Decode(dst); err != nil {
		a.reply(w, r, 0, nil, err)
		return false
	}
	if decoder.Decode(new(any)) != io.EOF {
		a.reply(w, r, 0, nil, domErrors.ErrInvalid)
		return false
	}
	return true
}

func (a *API) reply(w nethttp.ResponseWriter, r *nethttp.Request, status int, data any, err error) {
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		var statusCode int
		switch {
		case errors.Is(err, domErrors.ErrNotFound):
			statusCode = nethttp.StatusNotFound
		case errors.Is(err, domErrors.ErrConflict):
			statusCode = nethttp.StatusConflict
		case errors.Is(err, domErrors.ErrUnavailable):
			statusCode = nethttp.StatusServiceUnavailable
		default:
			statusCode = nethttp.StatusUnprocessableEntity
		}
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if status == 0 {
		status = nethttp.StatusOK
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
