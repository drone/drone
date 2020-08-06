// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package rpc2

import (
	"net/http"

	"github.com/drone/drone/operator/manager"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// Server wraps the chi Router in a custom type for wire
// injection purposes.
type Server http.Handler

// NewServer returns a new rpc server that enables remote
// interaction with the build controller using the http transport.
func NewServer(manager manager.BuildManager, secret string) Server {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.NoCache)
	r.Use(authorization(secret))
	r.Post("/nodes/:machine", HandleJoin())
	r.Delete("/nodes/:machine", HandleLeave())
	r.Post("/ping", HandlePing())
	r.Post("/stage", HandleRequest(manager))
	r.Post("/stage/{stage}", HandleAccept(manager))
	r.Get("/stage/{stage}", HandleInfo(manager))
	r.Put("/stage/{stage}", HandleUpdateStage(manager))
	r.Put("/step/{step}", HandleUpdateStep(manager))
	r.Post("/build/{build}/watch", HandleWatch(manager))
	r.Post("/step/{step}/logs/batch", HandleLogBatch(manager))
	r.Post("/step/{step}/logs/upload", HandleLogUpload(manager))
	return Server(r)
}

func authorization(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// prevents system administrators from accidentally
			// exposing drone without credentials.
			if token == "" {
				w.WriteHeader(403)
			} else if token == r.Header.Get("X-Drone-Token") {
				next.ServeHTTP(w, r)
			} else {
				w.WriteHeader(401)
			}
		})
	}
}
