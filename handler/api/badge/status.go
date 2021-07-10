// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package badge

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/drone/drone/core"

	"github.com/go-chi/chi"
)

// Handler returns an http.HandlerFunc that writes an svg status
// badge to the response.
func Handler(
	repos core.RepositoryStore,
	builds core.BuildStore,
	stages core.StageStore,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		namespace := chi.URLParam(r, "owner")
		name := chi.URLParam(r, "name")
		ref := r.FormValue("ref")
		branch := r.FormValue("branch")
		stepName := chi.URLParam(r, "step")

		if branch != "" {
			ref = "refs/heads/" + branch
		}

		// an SVG response is always served, even when error, so
		// we can go ahead and set the content type appropriately.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate, value")
		w.Header().Set("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.Header().Set("Content-Type", "image/svg+xml")

		repo, err := repos.FindName(r.Context(), namespace, name)
		if err != nil {
			io.WriteString(w, badgeNone)
			return
		}

		if ref == "" {
			ref = fmt.Sprintf("refs/heads/%s", repo.Branch)
		}
		build, err := builds.FindRef(r.Context(), repo.ID, ref)
		if err != nil {
			io.WriteString(w, badgeNone)
			return
		}

		if stepName != "" {
			steps, err := stages.ListSteps(r.Context(), build.ID)
			if err != nil {
				io.WriteString(w, badgeNone)
				return
			}

			// try to find the step by name
			stepID := -1
			for k, step := range steps {
				if step.Name == stepName {
					stepID = k
					break
				}
			}

			if stepID != -1 {
				switch steps[stepID].Status {
				case core.StatusPending, core.StatusRunning, core.StatusBlocked:
					io.WriteString(w, badgeStarted)
				case core.StatusPassing:
					io.WriteString(w, badgeSuccess)
				case core.StatusError:
					io.WriteString(w, badgeError)
				default:
					io.WriteString(w, badgeFailure)
				}
				return
			} else {
				io.WriteString(w, badgeNone)
				return
			}
		}

		switch build.Status {
		case core.StatusPending, core.StatusRunning, core.StatusBlocked:
			io.WriteString(w, badgeStarted)
		case core.StatusPassing:
			io.WriteString(w, badgeSuccess)
		case core.StatusError:
			io.WriteString(w, badgeError)
		default:
			io.WriteString(w, badgeFailure)
		}
	}
}
