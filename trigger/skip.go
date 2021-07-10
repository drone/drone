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
	"strings"

	"github.com/drone/drone-yaml/yaml"
	"github.com/drone/drone/core"
	"github.com/drone/go-scm/scm"
)

func skipBranch(document *yaml.Pipeline, branch string) bool {
	return !document.Trigger.Branch.Match(branch)
}

func skipRef(document *yaml.Pipeline, ref string) bool {
	return !document.Trigger.Ref.Match(ref)
}

func skipEvent(document *yaml.Pipeline, event string) bool {
	return !document.Trigger.Event.Match(event)
}

func skipAction(document *yaml.Pipeline, action string) bool {
	return !document.Trigger.Action.Match(action)
}

// skipEventAction implements event specific logic where the standard "subscribe to all" event logic is not
// appropriate.
func skipEventAction(document *yaml.Pipeline, event string, action string) bool {
	switch event {
	case core.EventPullRequest:
		return skipPullRequestEval(&document.Trigger.Action, action)
	}

	return false
}

func skipInstance(document *yaml.Pipeline, instance string) bool {
	return !document.Trigger.Instance.Match(instance)
}

func skipTarget(document *yaml.Pipeline, env string) bool {
	return !document.Trigger.Target.Match(env)
}

func skipRepo(document *yaml.Pipeline, repo string) bool {
	return !document.Trigger.Repo.Match(repo)
}

func skipCron(document *yaml.Pipeline, cron string) bool {
	return !document.Trigger.Cron.Match(cron)
}

func skipMessage(hook *core.Hook) bool {
	switch {
	case hook.Event == core.EventTag:
		return false
	case hook.Event == core.EventCron:
		return false
	case hook.Event == core.EventCustom:
		return false
	case hook.Event == core.EventPromote:
		return false
	case hook.Event == core.EventRollback:
		return false
	case skipMessageEval(hook.Message):
		return true
	case skipMessageEval(hook.Title):
		return true
	default:
		return false
	}
}

func skipMessageEval(str string) bool {
	lower := strings.ToLower(str)
	switch {
	case strings.Contains(lower, "[ci skip]"),
		strings.Contains(lower, "[skip ci]"),
		strings.Contains(lower, "***no_ci***"):
		return true
	default:
		return false
	}
}

// skipPullRequestEval determines whether or not to skip pull requests.
//
// Pull requests have a special behaviour in that unless the "close" event is deliberately opted in to via "include"
// it should not be included. This allows users to consume this event class, but does not break BC with users who
// have pipelines defined before this event was introduced.
func skipPullRequestEval(condition *yaml.Condition, action string) bool {
	// If there are conditions and those conditions include this action, allow the pipeline to continue.
	if len(condition.Include) > 0 && condition.Includes(action) {
		return false
	}

	// Verify the event is not deliberately excluded
	if condition.Excludes(action) {
		return true
	}

	// If there are no includes and execludes for this event, see if it matches the defaults:
	if len(condition.Include) == 0 {
		switch action {
		case scm.ActionOpen.String(), scm.ActionSync.String():
			return false
		}
	}

	// If there are still conditions but they do not match this action, the pipeline should be skipped.
	return true
}

// func skipPaths(document *config.Config, paths []string) bool {
// 	switch {
// 	// changed files are only returned for push and pull request
// 	// events. If the list of changed files is empty the system will
// 	// force-run all pipelines and pipeline steps
// 	case len(paths) == 0:
// 		return false
// 	// github returns a maximum of 300 changed files from the
// 	// api response. If there are 300+ changed files the system
// 	// will force-run all pipelines and pipeline steps.
// 	case len(paths) >= 300:
// 		return false
// 	default:
// 		return !document.Trigger.Paths.MatchAny(paths)
// 	}
// }
