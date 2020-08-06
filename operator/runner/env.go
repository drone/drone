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

package runner

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/drone/drone/core"
)

func systemEnviron(system *core.System) map[string]string {
	return map[string]string{
		"CI":                    "true",
		"DRONE":                 "true",
		"DRONE_SYSTEM_PROTO":    system.Proto,
		"DRONE_SYSTEM_HOST":     system.Host,
		"DRONE_SYSTEM_HOSTNAME": system.Host,
		"DRONE_SYSTEM_VERSION":  fmt.Sprint(system.Version),
	}
}

func agentEnviron(runner *Runner) map[string]string {
	return map[string]string{
		"DRONE_MACHINE":         runner.Machine,
		"DRONE_RUNNER_HOST":     runner.Machine,
		"DRONE_RUNNER_HOSTNAME": runner.Machine,
		"DRONE_RUNNER_PLATFORM": runner.Platform,
	}
}

func repoEnviron(repo *core.Repository) map[string]string {
	return map[string]string{
		"DRONE_REPO":            repo.Slug,
		"DRONE_REPO_SCM":        repo.SCM,
		"DRONE_REPO_OWNER":      repo.Namespace,
		"DRONE_REPO_NAMESPACE":  repo.Namespace,
		"DRONE_REPO_NAME":       repo.Name,
		"DRONE_REPO_LINK":       repo.Link,
		"DRONE_REPO_BRANCH":     repo.Branch,
		"DRONE_REMOTE_URL":      repo.HTTPURL,
		"DRONE_GIT_HTTP_URL":    repo.HTTPURL,
		"DRONE_GIT_SSH_URL":     repo.SSHURL,
		"DRONE_REPO_VISIBILITY": repo.Visibility,
		"DRONE_REPO_PRIVATE":    fmt.Sprint(repo.Private),

		//
		// these are legacy configuration parameters for backward
		// compatibility with drone 0.8.
		//
		"CI_REPO":         repo.Slug,
		"CI_REPO_NAME":    repo.Slug,
		"CI_REPO_LINK":    repo.Link,
		"CI_REPO_REMOTE":  repo.HTTPURL,
		"CI_REMOTE_URL":   repo.HTTPURL,
		"CI_REPO_PRIVATE": fmt.Sprint(repo.Private),
	}
}

func stageEnviron(stage *core.Stage) map[string]string {
	return map[string]string{
		"DRONE_STAGE_KIND":       "pipeline",
		"DRONE_STAGE_NAME":       stage.Name,
		"DRONE_STAGE_NUMBER":     fmt.Sprint(stage.Number),
		"DRONE_STAGE_MACHINE":    stage.Machine,
		"DRONE_STAGE_OS":         stage.OS,
		"DRONE_STAGE_ARCH":       stage.Arch,
		"DRONE_STAGE_VARIANT":    stage.Variant,
		"DRONE_STAGE_DEPENDS_ON": strings.Join(stage.DependsOn, ","),
	}
}

func buildEnviron(build *core.Build) map[string]string {
	env := map[string]string{
		"DRONE_BRANCH":               build.Target,
		"DRONE_SOURCE_BRANCH":        build.Source,
		"DRONE_TARGET_BRANCH":        build.Target,
		"DRONE_COMMIT":               build.After,
		"DRONE_COMMIT_SHA":           build.After,
		"DRONE_COMMIT_BEFORE":        build.Before,
		"DRONE_COMMIT_AFTER":         build.After,
		"DRONE_COMMIT_REF":           build.Ref,
		"DRONE_COMMIT_BRANCH":        build.Target,
		"DRONE_COMMIT_LINK":          build.Link,
		"DRONE_COMMIT_MESSAGE":       build.Message,
		"DRONE_COMMIT_AUTHOR":        build.Author,
		"DRONE_COMMIT_AUTHOR_EMAIL":  build.AuthorEmail,
		"DRONE_COMMIT_AUTHOR_AVATAR": build.AuthorAvatar,
		"DRONE_COMMIT_AUTHOR_NAME":   build.AuthorName,
		"DRONE_BUILD_NUMBER":         fmt.Sprint(build.Number),
		"DRONE_BUILD_EVENT":          build.Event,
		"DRONE_BUILD_ACTION":         build.Action,
		"DRONE_BUILD_CREATED":        fmt.Sprint(build.Created),
		"DRONE_BUILD_STARTED":        fmt.Sprint(build.Started),
		"DRONE_BUILD_FINISHED":       fmt.Sprint(build.Finished),
		"DRONE_DEPLOY_TO":            build.Deploy,

		//
		// these are legacy configuration parameters for backward
		// compatibility with drone 0.8.
		//
		"CI_BUILD_NUMBER":         fmt.Sprint(build.Number),
		"CI_PARENT_BUILD_NUMBER":  fmt.Sprint(build.Parent),
		"CI_BUILD_CREATED":        fmt.Sprint(build.Created),
		"CI_BUILD_STARTED":        fmt.Sprint(build.Started),
		"CI_BUILD_FINISHED":       fmt.Sprint(build.Finished),
		"CI_BUILD_STATUS":         build.Status,
		"CI_BUILD_EVENT":          build.Event,
		"CI_BUILD_LINK":           build.Link,
		"CI_BUILD_TARGET":         build.Deploy,
		"CI_COMMIT_SHA":           build.After,
		"CI_COMMIT_REF":           build.Ref,
		"CI_COMMIT_BRANCH":        build.Target,
		"CI_COMMIT_MESSAGE":       build.Message,
		"CI_COMMIT_AUTHOR":        build.Author,
		"CI_COMMIT_AUTHOR_NAME":   build.AuthorName,
		"CI_COMMIT_AUTHOR_EMAIL":  build.AuthorEmail,
		"CI_COMMIT_AUTHOR_AVATAR": build.AuthorAvatar,
	}
	if strings.HasPrefix(build.Ref, "refs/tags/") {
		env["DRONE_TAG"] = strings.TrimPrefix(build.Ref, "refs/tags/")
	}
	if build.Event == core.EventPullRequest {
		env["DRONE_PULL_REQUEST"] = re.FindString(build.Ref)
	}
	return env
}

func linkEnviron(repo *core.Repository, build *core.Build, system *core.System) map[string]string {
	return map[string]string{
		"DRONE_BUILD_LINK": fmt.Sprintf(
			"%s://%s/%s/%d",
			system.Proto,
			system.Host,
			repo.Slug,
			build.Number,
		),
	}
}

// regular expression to extract the pull request number
// from the git ref (e.g. refs/pulls/{d}/head)
var re = regexp.MustCompile("\\d+")

// helper function combines one or more maps of environment
// variables into a single map.
func combineEnviron(env ...map[string]string) map[string]string {
	c := map[string]string{}
	for _, e := range env {
		for k, v := range e {
			c[k] = v
		}
	}
	return c
}
