// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package system

import (
	"net/http"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/logger"
)

type (
	users struct {
		Total int64 `json:"total"`
	}

	repos struct {
		Active int64 `json:"active"`
	}

	builds struct {
		Pending int   `json:"pending"`
		Running int   `json:"running"`
		Total   int64 `json:"total"`
	}

	events struct {
		Subscribers int `json:"subscribers"`
	}

	streams struct {
		Subscribers int `json:"subscribers"`
		Channels    int `json:"channels"`
	}

	platform struct {
		Subscribers int    `json:"subscribers"`
		OS          string `json:"os"`
		Arch        string `json:"arch"`
		Variant     string `json:"variant"`
		Kernel      string `json:"kernel"`
		Pending     int    `json:"pending"`
		Running     int    `json:"running"`
	}

	stats struct {
		Users     users         `json:"users"`
		Repos     repos         `json:"repos"`
		Builds    builds        `json:"builds"`
		Pipelines []*platform   `json:"pipelines"`
		Events    events        `json:"events"`
		Streams   map[int64]int `json:"streams"`
		Watchers  map[int64]int `json:"watchers"`
	}
)

// HandleStats returns an http.HandlerFunc that writes a
// json-encoded list of system stats to the response body.
func HandleStats(
	builds core.BuildStore,
	stages core.StageStore,
	users core.UserStore,
	repos core.RepositoryStore,
	bus core.Pubsub,
	streams core.LogStream,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ctx = r.Context()
		var err error

		//
		// User Stats
		//

		stats := &stats{}
		stats.Users.Total, err = users.Count(ctx)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).WithError(err).
				Warnln("stats: cannot get user count")
			return
		}

		//
		// Repo Stats
		//

		stats.Repos.Active, err = repos.Count(ctx)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).WithError(err).
				Warnln("stats: cannot get repo count")
			return
		}

		//
		// Build Stats
		//

		stats.Builds.Total, err = builds.Count(ctx)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).WithError(err).
				Warnln("stats: cannot get build count")
			return
		}
		buildsPending, err := builds.Pending(ctx)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).WithError(err).
				Warnln("stats: cannot get pending build count")
			return
		}
		buildsRunning, err := builds.Running(ctx)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).WithError(err).
				Warnln("stats: cannot get running build count")
			return
		}
		stats.Builds.Pending = len(buildsPending)
		stats.Builds.Running = len(buildsRunning)

		//
		// Queue Stats
		//

		incomplete, err := stages.ListIncomplete(ctx)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).WithError(err).
				Warnln("stats: cannot get pending stage count")
			return
		}
		platforms := newPlatformList()
		aggregatePlatformStats(platforms, incomplete)
		stats.Pipelines = platforms

		//
		// Event Stats
		//

		stats.Events.Subscribers = bus.Subscribers()

		//
		// Stream Stats
		//

		stats.Streams = streams.Info(ctx).Streams

		render.JSON(w, stats, 200)
	}
}

// platform statistics are returned in a fixed array. these
// are pointers to the platform index in the array.
const (
	linuxArm6 int = iota
	linuxArm7
	linuxArm8
	linuxArm9
	linuxAmd64
	linuxPpc64le
	windows1709
	windows1803
	windows1809
)

// helper function returns a list of all platforms
// and variants currently supported by core.
func newPlatformList() []*platform {
	platforms := [9]*platform{}
	platforms[linuxArm6] = &platform{OS: "linux", Arch: "arm", Variant: "v6"}
	platforms[linuxArm7] = &platform{OS: "linux", Arch: "arm", Variant: "v7"}
	platforms[linuxArm8] = &platform{OS: "linux", Arch: "arm64", Variant: "v8"}
	platforms[linuxArm9] = &platform{OS: "linux", Arch: "arm", Variant: "v9"}
	platforms[linuxAmd64] = &platform{OS: "linux", Arch: "amd64"}
	platforms[linuxPpc64le] = &platform{OS: "linux", Arch: "ppc64le"}
	platforms[windows1709] = &platform{OS: "windows", Arch: "arm64", Kernel: "1709"}
	platforms[windows1803] = &platform{OS: "windows", Arch: "arm64", Kernel: "1803"}
	platforms[windows1809] = &platform{OS: "windows", Arch: "arm64", Kernel: "1809"}
	return platforms[:]
}

// helper function counts the number of running and
// pending stages by os, architecture, and variant.
func aggregatePlatformStats(platforms []*platform, stages []*core.Stage) {
	for _, stage := range stages {
		var index int
		switch {
		case stage.OS == "windows" && stage.Kernel == "1709":
			index = windows1709
		case stage.OS == "windows" && stage.Kernel == "1803":
			index = windows1803
		case stage.OS == "windows" && stage.Kernel == "1809":
			index = windows1809
		case stage.OS == "windows":
			// default to 1803 when no variant specified
			index = windows1809
		case stage.Arch == "arm" && stage.Variant == "v6":
			index = linuxArm6
		case stage.Arch == "arm" && stage.Variant == "v7":
			index = linuxArm7
		case stage.Arch == "arm" && stage.Variant == "v9":
			index = linuxArm9
		case stage.Arch == "arm":
			// default to arm7 when no variant specified
			index = linuxArm7
		case stage.Arch == "arm64" && stage.Variant == "v8":
			index = linuxArm8
		case stage.Arch == "arm64":
			// default to arm8 when arm64
			index = linuxArm8
		case stage.Arch == "ppc64le":
			index = linuxPpc64le
		default:
			index = linuxAmd64
			continue
		}

		switch stage.Status {
		case core.StatusPending:
			platforms[index].Pending++
		case core.StatusRunning:
			platforms[index].Running++
		}
	}
}
