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

package trigger

import (
	"context"
	"runtime/debug"
	"strings"
	"time"

	"github.com/drone/drone-yaml/yaml"
	"github.com/drone/drone-yaml/yaml/converter"
	"github.com/drone/drone-yaml/yaml/linter"
	"github.com/drone/drone-yaml/yaml/signer"

	"github.com/drone/drone/core"
	"github.com/drone/drone/trigger/dag"

	"github.com/sirupsen/logrus"
)

type triggerer struct {
	canceler core.Canceler
	config   core.ConfigService
	convert  core.ConvertService
	commits  core.CommitService
	status   core.StatusService
	builds   core.BuildStore
	sched    core.Scheduler
	repos    core.RepositoryStore
	users    core.UserStore
	validate core.ValidateService
	hooks    core.WebhookSender
}

// New returns a new build triggerer.
func New(
	canceler core.Canceler,
	config core.ConfigService,
	convert core.ConvertService,
	commits core.CommitService,
	status core.StatusService,
	builds core.BuildStore,
	sched core.Scheduler,
	repos core.RepositoryStore,
	users core.UserStore,
	validate core.ValidateService,
	hooks core.WebhookSender,
) core.Triggerer {
	return &triggerer{
		canceler: canceler,
		config:   config,
		convert:  convert,
		commits:  commits,
		status:   status,
		builds:   builds,
		sched:    sched,
		repos:    repos,
		users:    users,
		validate: validate,
		hooks:    hooks,
	}
}

func (t *triggerer) Trigger(ctx context.Context, repo *core.Repository, base *core.Hook) (*core.Build, error) {
	logger := logrus.WithFields(
		logrus.Fields{
			"repo":   repo.Slug,
			"ref":    base.Ref,
			"event":  base.Event,
			"commit": base.After,
		},
	)

	logger.Debugln("trigger: received")
	defer func() {
		// taking the paranoid approach to recover from
		// a panic that should absolutely never happen.
		if r := recover(); r != nil {
			logger.Errorf("runner: unexpected panic: %s", r)
			debug.PrintStack()
		}
	}()

	user, err := t.users.Find(ctx, repo.UserID)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("trigger: cannot find repository owner")
		return nil, err
	}

	if user.Active == false {
		logger.Infoln("trigger: skipping hook. repository owner is inactive")
		return nil, nil
	}

	// if the commit message is not included we should
	// make an optional API call to the version control
	// system to augment the available information.
	if base.Message == "" && base.After != "" {
		commit, err := t.commits.Find(ctx, user, repo.Slug, base.After)
		if err == nil && commit != nil {
			base.Message = commit.Message
			if base.AuthorEmail == "" {
				base.AuthorEmail = commit.Author.Email
			}
			if base.AuthorName == "" {
				base.AuthorName = commit.Author.Name
			}
			if base.AuthorAvatar == "" {
				base.AuthorAvatar = commit.Author.Avatar
			}
		}
	}
	
	// do skip trigger after base hook has been filled by commit message
	if skipMessage(base) {
		logger.Infoln("trigger: skipping hook. found skip directive")
		return nil, nil
	}
	if base.Event == core.EventPullRequest {
		if repo.IgnorePulls {
			logger.Infoln("trigger: skipping hook. project ignores pull requests")
			return nil, nil
		}
		if repo.IgnoreForks && !strings.EqualFold(base.Fork, repo.Slug) {
			logger.Infoln("trigger: skipping hook. project ignores forks")
			return nil, nil
		}
	}

	// // some tag hooks provide the tag but do not provide the sha.
	// // this may be important if we want to fetch the .drone.yml
	// if base.After == "" && base.Event == core.EventTag {
	// 	tag, _, err := t.client.Git.FindTag(ctx, repo.Slug, base.Ref)
	// 	if err != nil {
	// 		logger.Error().Err(err).
	// 			Msg("cannot find tag")
	// 		return nil, err
	// 	}
	// 	base.After = tag.Sha
	// }

	// TODO: do a better job of documenting this
	// obj := base.After
	// if len(obj) == 0 {
	// 	if strings.HasPrefix(base.Ref, "refs/pull/") {
	// 		obj = base.Target
	// 	} else {
	// 		obj = base.Ref
	// 	}
	// }
	tmpBuild := &core.Build{
		RepoID:  repo.ID,
		Trigger: base.Trigger,
		Parent:  base.Parent,
		Status:  core.StatusPending,
		Event:   base.Event,
		Action:  base.Action,
		Link:    base.Link,
		// Timestamp:    base.Timestamp,
		Title:        base.Title,
		Message:      base.Message,
		Before:       base.Before,
		After:        base.After,
		Ref:          base.Ref,
		Fork:         base.Fork,
		Source:       base.Source,
		Target:       base.Target,
		Author:       base.Author,
		AuthorName:   base.AuthorName,
		AuthorEmail:  base.AuthorEmail,
		AuthorAvatar: base.AuthorAvatar,
		Params:       base.Params,
		Cron:         base.Cron,
		Deploy:       base.Deployment,
		DeployID:     base.DeploymentID,
		Debug:        base.Debug,
		Sender:       base.Sender,
		Created:      time.Now().Unix(),
		Updated:      time.Now().Unix(),
	}
	req := &core.ConfigArgs{
		User:  user,
		Repo:  repo,
		Build: tmpBuild,
	}
	raw, err := t.config.Find(ctx, req)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("trigger: cannot find yaml")
		return nil, err
	}

	raw, err = t.convert.Convert(ctx, &core.ConvertArgs{
		User:   user,
		Repo:   repo,
		Build:  tmpBuild,
		Config: raw,
	})
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("trigger: cannot convert yaml")
		return t.createBuildError(ctx, repo, base, err.Error())
	}

	// this code is temporarily in place to detect and convert
	// the legacy yaml configuration file to the new format.
	raw.Data, err = converter.ConvertString(raw.Data, converter.Metadata{
		Filename: repo.Config,
		URL:      repo.Link,
		Ref:      base.Ref,
	})
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("trigger: cannot convert yaml")
		return t.createBuildError(ctx, repo, base, err.Error())
	}

	manifest, err := yaml.ParseString(raw.Data)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("trigger: cannot parse yaml")
		return t.createBuildError(ctx, repo, base, err.Error())
	}

	verr := t.validate.Validate(ctx, &core.ValidateArgs{
		User:   user,
		Repo:   repo,
		Build:  tmpBuild,
		Config: raw,
	})
	switch verr {
	case core.ErrValidatorBlock:
		logger.Debugln("trigger: yaml validation error: block pipeline")
	case core.ErrValidatorSkip:
		logger.Debugln("trigger: yaml validation error: skip pipeline")
		return nil, nil
	default:
		if verr != nil {
			logger = logger.WithError(err)
			logger.Warnln("trigger: yaml validation error")
			return t.createBuildError(ctx, repo, base, verr.Error())
		}
	}

	err = linter.Manifest(manifest, repo.Trusted)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("trigger: yaml linting error")
		return t.createBuildError(ctx, repo, base, err.Error())
	}

	verified := true
	if repo.Protected && base.Trigger == core.TriggerHook {
		key := signer.KeyString(repo.Secret)
		val := []byte(raw.Data)
		verified, _ = signer.Verify(val, key)
	}
	// if pipeline validation failed with a block error, the
	// pipeline verification should be set to false, which will
	// force manual review and approval.
	if verr == core.ErrValidatorBlock {
		verified = false
	}

	// var paths []string
	// paths, err := listChanges(t.client, repo, base)
	// if err != nil {
	// 	logger.Warn().Err(err).
	// 		Msg("cannot fetch changeset")
	// }

	var matched []*yaml.Pipeline
	var dag = dag.New()
	for _, document := range manifest.Resources {
		pipeline, ok := document.(*yaml.Pipeline)
		if !ok {
			continue
		}
		// TODO add repo
		// TODO add instance
		// TODO add target
		// TODO add ref
		name := pipeline.Name
		if name == "" {
			name = "default"
		}
		node := dag.Add(pipeline.Name, pipeline.DependsOn...)
		node.Skip = true

		if skipBranch(pipeline, base.Target) {
			logger = logger.WithField("pipeline", pipeline.Name)
			logger.Infoln("trigger: skipping pipeline, does not match branch")
		} else if skipEvent(pipeline, base.Event) {
			logger = logger.WithField("pipeline", pipeline.Name)
			logger.Infoln("trigger: skipping pipeline, does not match event")
		} else if skipAction(pipeline, base.Action) {
			logger = logger.WithField("pipeline", pipeline.Name).WithField("action", base.Action)
			logger.Infoln("trigger: skipping pipeline, does not match action")
		} else if skipRef(pipeline, base.Ref) {
			logger = logger.WithField("pipeline", pipeline.Name)
			logger.Infoln("trigger: skipping pipeline, does not match ref")
		} else if skipRepo(pipeline, repo.Slug) {
			logger = logger.WithField("pipeline", pipeline.Name)
			logger.Infoln("trigger: skipping pipeline, does not match repo")
		} else if skipTarget(pipeline, base.Deployment) {
			logger = logger.WithField("pipeline", pipeline.Name)
			logger.Infoln("trigger: skipping pipeline, does not match deploy target")
		} else if skipCron(pipeline, base.Cron) {
			logger = logger.WithField("pipeline", pipeline.Name)
			logger.Infoln("trigger: skipping pipeline, does not match cron job")
		} else {
			matched = append(matched, pipeline)
			node.Skip = false
		}
	}

	if dag.DetectCycles() {
		return t.createBuildError(ctx, repo, base, "Error: Dependency cycle detected in Pipeline")
	}

	if len(matched) == 0 {
		logger.Infoln("trigger: skipping build, no matching pipelines")
		return nil, nil
	}

	repo, err = t.repos.Increment(ctx, repo)
	if err != nil {
		logger = logger.WithError(err)
		logger.Errorln("trigger: cannot increment build sequence")
		return nil, err
	}

	build := &core.Build{
		RepoID:  repo.ID,
		Trigger: base.Trigger,
		Number:  repo.Counter,
		Parent:  base.Parent,
		Status:  core.StatusPending,
		Event:   base.Event,
		Action:  base.Action,
		Link:    base.Link,
		// Timestamp:    base.Timestamp,
		Title:        trunc(base.Title, 2000),
		Message:      trunc(base.Message, 2000),
		Before:       base.Before,
		After:        base.After,
		Ref:          base.Ref,
		Fork:         base.Fork,
		Source:       base.Source,
		Target:       base.Target,
		Author:       base.Author,
		AuthorName:   base.AuthorName,
		AuthorEmail:  base.AuthorEmail,
		AuthorAvatar: base.AuthorAvatar,
		Params:       base.Params,
		Deploy:       base.Deployment,
		DeployID:     base.DeploymentID,
		Debug:        base.Debug,
		Sender:       base.Sender,
		Cron:         base.Cron,
		Created:      time.Now().Unix(),
		Updated:      time.Now().Unix(),
	}

	stages := make([]*core.Stage, len(matched))
	for i, match := range matched {
		onSuccess := match.Trigger.Status.Match(core.StatusPassing)
		onFailure := match.Trigger.Status.Match(core.StatusFailing)
		if len(match.Trigger.Status.Include)+len(match.Trigger.Status.Exclude) == 0 {
			onFailure = false
		}

		stage := &core.Stage{
			RepoID:    repo.ID,
			Number:    i + 1,
			Name:      match.Name,
			Kind:      match.Kind,
			Type:      match.Type,
			OS:        match.Platform.OS,
			Arch:      match.Platform.Arch,
			Variant:   match.Platform.Variant,
			Kernel:    match.Platform.Version,
			Limit:     match.Concurrency.Limit,
			LimitRepo: int(repo.Throttle),
			Status:    core.StatusWaiting,
			DependsOn: match.DependsOn,
			OnSuccess: onSuccess,
			OnFailure: onFailure,
			Labels:    match.Node,
			Created:   time.Now().Unix(),
			Updated:   time.Now().Unix(),
		}
		if stage.Kind == "pipeline" && stage.Type == "" {
			stage.Type = "docker"
		}
		if stage.OS == "" {
			stage.OS = "linux"
		}
		if stage.Arch == "" {
			stage.Arch = "amd64"
		}

		if stage.Name == "" {
			stage.Name = "default"
		}
		if verified == false {
			stage.Status = core.StatusBlocked
		} else if len(stage.DependsOn) == 0 {
			stage.Status = core.StatusPending
		}
		stages[i] = stage
	}

	for _, stage := range stages {
		// here we re-work the dependencies for the stage to
		// account for the fact that some steps may be skipped
		// and may otherwise break the dependency chain.
		stage.DependsOn = dag.Dependencies(stage.Name)

		// if the stage is pending dependencies, but those
		// dependencies are skipped, the stage can be executed
		// immediately.
		if stage.Status == core.StatusWaiting &&
			len(stage.DependsOn) == 0 {
			stage.Status = core.StatusPending
		}
	}

	err = t.builds.Create(ctx, build, stages)
	if err != nil {
		logger = logger.WithError(err)
		logger.Errorln("trigger: cannot create build")
		return nil, err
	}

	err = t.status.Send(ctx, user, &core.StatusInput{
		Repo:  repo,
		Build: build,
	})
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("trigger: cannot create status")
	}

	for _, stage := range stages {
		if stage.Status != core.StatusPending {
			continue
		}
		err = t.sched.Schedule(ctx, stage)
		if err != nil {
			logger = logger.WithError(err)
			logger.Errorln("trigger: cannot enqueue build")
			return nil, err
		}
	}

	payload := &core.WebhookData{
		Event:  core.WebhookEventBuild,
		Action: core.WebhookActionCreated,
		User:   user,
		Repo:   repo,
		Build:  build,
	}
	err = t.hooks.Send(ctx, payload)
	if err != nil {
		logger = logger.WithError(err)
		logger.Warnln("trigger: cannot send webhook")
	}

	if repo.CancelPush && build.Event == core.EventPush ||
		repo.CancelPulls && build.Event == core.EventPullRequest {
		go t.canceler.CancelPending(ctx, repo, build)
	}

	// err = t.hooks.SendEndpoint(ctx, payload, repo.Endpoints.Webhook)
	// if err != nil {
	// 	logger.Warn().Err(err).
	// 		Int64("build", build.Number).
	// 		Msg("cannot send user-defined webhook")
	// }

	// // we should only synchronize the cronjob list on push
	// // events to the default branch.
	// if build.Event == core.EventPush &&
	// 	build.Target == repo.Branch {
	// 	err = t.cron.Sync(ctx, repo, manifest)
	// 	if err != nil {
	// 		logger.Warn().Err(err).
	// 			Msg("cannot sync cronjobs")
	// 	}
	// }

	return build, nil
}

func trunc(s string, i int) string {
	runes := []rune(s)
	if len(runes) > i {
		return string(runes[:i])
	}
	return s
}

func (t *triggerer) createBuildError(ctx context.Context, repo *core.Repository, base *core.Hook, message string) (*core.Build, error) {
	repo, err := t.repos.Increment(ctx, repo)
	if err != nil {
		return nil, err
	}

	build := &core.Build{
		RepoID: repo.ID,
		Number: repo.Counter,
		Parent: base.Parent,
		Status: core.StatusError,
		Error:  message,
		Event:  base.Event,
		Action: base.Action,
		Link:   base.Link,
		// Timestamp:    base.Timestamp,
		Title:        base.Title,
		Message:      base.Message,
		Before:       base.Before,
		After:        base.After,
		Ref:          base.Ref,
		Fork:         base.Fork,
		Source:       base.Source,
		Target:       base.Target,
		Author:       base.Author,
		AuthorName:   base.AuthorName,
		AuthorEmail:  base.AuthorEmail,
		AuthorAvatar: base.AuthorAvatar,
		Deploy:       base.Deployment,
		DeployID:     base.DeploymentID,
		Debug:        base.Debug,
		Sender:       base.Sender,
		Created:      time.Now().Unix(),
		Updated:      time.Now().Unix(),
		Started:      time.Now().Unix(),
		Finished:     time.Now().Unix(),
	}

	err = t.builds.Create(ctx, build, nil)
	return build, err
}

// func shouldBlock(repo *core.Repository, build *core.Build) bool {
// 	switch {
// 	case repo.Hooks.Promote == core.HookBlock && build.Event == core.EventPromote:
// 		return true
// 	case repo.Hooks.Rollback == core.HookBlock && build.Event == core.EventRollback:
// 		return true
// 	case repo.Hooks.Deploy == core.HookBlock && build.Event == core.EventRollback:
// 		return true
// 	case repo.Hooks.Pull == core.HookBlock && build.Event == core.EventPullRequest:
// 		return true
// 	case repo.Hooks.Push == core.HookBlock && build.Event == core.EventPush:
// 		return true
// 	case repo.Hooks.Tags == core.HookBlock && build.Event == core.EventTag:
// 		return true
// 	case repo.Hooks.Forks == core.HookBlock && build.Fork != repo.Slug:
// 		return true
// 	default:
// 		return false
// 	}
// }

// func skipHook(repo *core.Repository, build *core.Hook) bool {
// 	switch {
// 	case repo.Hooks.Promote == core.HookDisable && build.Event == core.EventPromote:
// 		return true
// 	case repo.Hooks.Rollback == core.HookDisable && build.Event == core.EventRollback:
// 		return true
// 	case repo.Hooks.Pull == core.HookDisable && build.Event == core.EventPullRequest:
// 		return true
// 	case repo.Hooks.Push == core.HookDisable && build.Event == core.EventPush:
// 		return true
// 	case repo.Hooks.Tags == core.HookDisable && build.Event == core.EventTag:
// 		return true
// 	default:
// 		return false
// 	}
// }

// func skipFork(repo *core.Repository, build *core.Hook) bool {
// 	return repo.Hooks.Forks == core.HookDisable && build.Fork != repo.Slug
// }
