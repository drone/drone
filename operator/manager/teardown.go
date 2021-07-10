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

package manager

import (
	"context"
	"encoding/json"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
	"github.com/drone/go-scm/scm"

	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
)

type teardown struct {
	Builds    core.BuildStore
	Events    core.Pubsub
	Logs      core.LogStream
	Scheduler core.Scheduler
	Repos     core.RepositoryStore
	Steps     core.StepStore
	Status    core.StatusService
	Stages    core.StageStore
	Users     core.UserStore
	Webhook   core.WebhookSender
}

func (t *teardown) do(ctx context.Context, stage *core.Stage) error {
	logger := logrus.WithField("stage.id", stage.ID)
	logger.Debugln("manager: stage is complete. teardown")

	build, err := t.Builds.Find(noContext, stage.BuildID)
	if err != nil {
		logger.WithError(err).Warnln("manager: cannot find the build")
		return err
	}

	logger = logger.WithFields(
		logrus.Fields{
			"build.number": build.Number,
			"build.id":     build.ID,
			"repo.id":      build.RepoID,
		},
	)

	repo, err := t.Repos.Find(noContext, build.RepoID)
	if err != nil {
		logger.WithError(err).Warnln("manager: cannot find the repository")
		return err
	}

	for _, step := range stage.Steps {
		if len(step.Error) > 500 {
			step.Error = step.Error[:500]
		}
		err := t.Steps.Update(noContext, step)
		if err != nil {
			logger.WithError(err).
				WithField("stage.status", stage.Status).
				WithField("step.name", step.Name).
				WithField("step.id", step.ID).
				Warnln("manager: cannot persist the step")
			return err
		}
	}

	if len(stage.Error) > 500 {
		stage.Error = stage.Error[:500]
	}

	stage.Updated = time.Now().Unix()
	err = t.Stages.Update(noContext, stage)
	if err != nil {
		logger.WithError(err).
			Warnln("manager: cannot update the stage")
		return err
	}

	for _, step := range stage.Steps {
		t.Logs.Delete(noContext, step.ID)
	}

	stages, err := t.Stages.ListSteps(noContext, build.ID)
	if err != nil {
		logger.WithError(err).Warnln("manager: cannot get stages")
		return err
	}

	//
	//
	//

	err = t.cancelDownstream(ctx, stages)
	if err != nil {
		logger.WithError(err).
			Errorln("manager: cannot cancel downstream builds")
		return err
	}

	err = t.scheduleDownstream(ctx, stage, stages)
	if err != nil {
		logger.WithError(err).
			Errorln("manager: cannot schedule downstream builds")
		return err
	}

	//
	//
	//

	if isBuildComplete(stages) == false {
		logger.Debugln("manager: build pending completion of additional stages")
		return nil
	}

	logger.Debugln("manager: build is finished, teardown")

	build.Status = core.StatusPassing
	build.Finished = time.Now().Unix()
	for _, sibling := range stages {
		if sibling.Status == core.StatusKilled {
			build.Status = core.StatusKilled
			break
		}
		if sibling.Status == core.StatusFailing {
			build.Status = core.StatusFailing
			break
		}
		if sibling.Status == core.StatusError {
			build.Status = core.StatusError
			break
		}
	}
	if build.Started == 0 {
		build.Started = build.Finished
	}

	err = t.Builds.Update(noContext, build)
	if err == db.ErrOptimisticLock {
		logger.WithError(err).
			Warnln("manager: build updated by another goroutine")
		return nil
	}
	if err != nil {
		logger.WithError(err).
			Warnln("manager: cannot update the build")
		return err
	}

	repo.Build = build
	repo.Build.Stages = stages
	data, _ := json.Marshal(repo)
	err = t.Events.Publish(noContext, &core.Message{
		Repository: repo.Slug,
		Visibility: repo.Visibility,
		Data:       data,
	})
	if err != nil {
		logger.WithError(err).
			Warnln("manager: cannot publish build event")
	}

	payload := &core.WebhookData{
		Event:  core.WebhookEventBuild,
		Action: core.WebhookActionUpdated,
		Repo:   repo,
		Build:  build,
	}

	err = t.Webhook.Send(noContext, payload)
	if err != nil {
		logger.WithError(err).Warnln("manager: cannot send global WebhookActionUpdated webhook")
	}

	payload = &core.WebhookData{
		Event:  core.WebhookEventBuild,
		Action: core.WebhookActionCompleted,
		Repo:   repo,
		Build:  build,
	}
	err = t.Webhook.Send(noContext, payload)
	if err != nil {
		logger.WithError(err).Warnln("manager: cannot send global WebhookActionCompleted webhook")
	}

	user, err := t.Users.Find(noContext, repo.UserID)
	if err != nil {
		logger.WithError(err).
			Warnln("manager: cannot find repository owner")

		// this error is insufficient to fail the function,
		// however, execution of the function should be halted
		// to prevent a nil pointer in subsequent operations.
		return nil
	}

	req := &core.StatusInput{
		Repo:  repo,
		Build: build,
	}
	err = t.Status.Send(noContext, user, req)
	if err != nil && err != scm.ErrNotSupported {
		logger.WithError(err).
			Warnln("manager: cannot publish status")
	}
	return nil
}

// cancelDownstream is a helper function that tests for
// downstream stages and cancels them based on the overall
// pipeline state.
func (t *teardown) cancelDownstream(
	ctx context.Context,
	stages []*core.Stage,
) error {
	failed := false
	for _, s := range stages {
		if s.IsFailed() {
			failed = true
		}
	}

	var errs error
	for _, s := range stages {
		if s.Status != core.StatusWaiting {
			continue
		}

		var skip bool
		if failed == true && s.OnFailure == false {
			skip = true
		}
		if failed == false && s.OnSuccess == false {
			skip = true
		}
		if skip == false {
			continue
		}

		if areDepsComplete(s, stages) == false {
			continue
		}

		logger := logrus.WithFields(
			logrus.Fields{
				"stage.id":         s.ID,
				"stage.on_success": s.OnSuccess,
				"stage.on_failure": s.OnFailure,
				"stage.is_failure": failed,
				"stage.depends_on": s.DependsOn,
			},
		)
		logger.Debugln("manager: skipping step")

		s.Status = core.StatusSkipped
		s.Started = time.Now().Unix()
		s.Stopped = time.Now().Unix()
		err := t.Stages.Update(noContext, s)
		if err == db.ErrOptimisticLock {
			t.resync(ctx, s)
			continue
		}
		if err != nil {
			logger.WithError(err).
				Warnln("manager: cannot update stage status")
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}

// scheduleDownstream is a helper function that tests for
// downstream stages and schedules stages if all dependencies
// and execution requirements are met.
func (t *teardown) scheduleDownstream(
	ctx context.Context,
	stage *core.Stage,
	stages []*core.Stage,
) error {

	var errs error
	for _, sibling := range stages {
		if sibling.Status == core.StatusWaiting {
			if len(sibling.DependsOn) == 0 {
				continue
			}

			// PROBLEM: isDep only checks the direct parent
			// i think ....
			// if isDep(stage, sibling) == false {
			// 	continue
			// }
			if areDepsComplete(sibling, stages) == false {
				continue
			}
			// if isLastDep(stage, sibling, stages) == false {
			// 	continue
			// }

			logger := logrus.WithFields(
				logrus.Fields{
					"stage.id":         sibling.ID,
					"stage.name":       sibling.Name,
					"stage.depends_on": sibling.DependsOn,
				},
			)
			logger.Debugln("manager: schedule next stage")

			sibling.Status = core.StatusPending
			sibling.Updated = time.Now().Unix()
			err := t.Stages.Update(noContext, sibling)
			if err == db.ErrOptimisticLock {
				t.resync(ctx, sibling)
				continue
			}
			if err != nil {
				logger.WithError(err).
					Warnln("manager: cannot update stage status")
				errs = multierror.Append(errs, err)
			}

			err = t.Scheduler.Schedule(noContext, sibling)
			if err != nil {
				logger.WithError(err).
					Warnln("manager: cannot schedule stage")
				errs = multierror.Append(errs, err)
			}
		}
	}
	return errs
}

// resync updates the stage from the database. Note that it does
// not update the Version field. This is by design. It prevents
// the current go routine from updating a stage that has been
// updated by another go routine.
func (t *teardown) resync(ctx context.Context, stage *core.Stage) error {
	updated, err := t.Stages.Find(ctx, stage.ID)
	if err != nil {
		return err
	}
	stage.Status = updated.Status
	stage.Error = updated.Error
	stage.ExitCode = updated.ExitCode
	stage.Machine = updated.Machine
	stage.Started = updated.Started
	stage.Stopped = updated.Stopped
	stage.Created = updated.Created
	stage.Updated = updated.Updated
	return nil
}
