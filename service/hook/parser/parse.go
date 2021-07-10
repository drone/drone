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

package parser

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/go-scm/scm"
)

// TODO(bradrydzewski): stash, push hook missing link
// TODO(bradrydzewski): stash, tag hook missing timestamp
// TODO(bradrydzewski): stash, tag hook missing commit message
// TODO(bradrydzewski): stash, tag hook missing link
// TODO(bradrydzewski): stash, pull request hook missing link
// TODO(bradrydzewski): stash, hooks missing repository clone http url
// TODO(bradrydzewski): stash, hooks missing repository clone ssh url
// TODO(bradrydzewski): stash, hooks missing repository html link

// TODO(bradrydzewski): gogs, push hook missing author avatar, using sender instead.
// TODO(bradrydzewski): gogs, pull request hook missing commit sha.
// TODO(bradrydzewski): gogs, tag hook missing commit sha.
// TODO(bradrydzewski): gogs, sender missing Name field.
// TODO(bradrydzewski): gogs, push hook missing repository html url

// TODO(bradrydzewski): gitea, push hook missing author avatar, using sender instead.
// TODO(bradrydzewski): gitea, tag hook missing commit sha.
// TODO(bradrydzewski): gitea, sender missing Name field.
// TODO(bradrydzewski): gitea, push hook missing repository html url

// TODO(bradrydzewski): bitbucket, pull request hook missing author email.
// TODO(bradrydzewski): bitbucket, hooks missing default repository branch.

// TODO(bradrydzewski): github, push hook timestamp is negative value.
// TODO(bradrydzewski): github, pull request message is empty

// represents a deleted ref in the github webhook.
const emptyCommit = "0000000000000000000000000000000000000000"

// this is intended for local testing and instructs the handler
// to print the contents of the hook to stdout.
var debugPrintHook = false

func init() {
	debugPrintHook, _ = strconv.ParseBool(
		os.Getenv("DRONE_DEBUG_DUMP_HOOK"),
	)
}

// New returns a new HookParser.
func New(client *scm.Client) core.HookParser {
	return &parser{client}
}

type parser struct {
	client *scm.Client
}

func (p *parser) Parse(req *http.Request, secretFunc func(string) string) (*core.Hook, *core.Repository, error) {
	if debugPrintHook {
		// if DRONE_DEBUG_DUMP_HOOK=true print the http.Request
		// headers and body to stdout.
		out, _ := httputil.DumpRequest(req, true)
		os.Stderr.Write(out)
	}

	// callback function provides the webhook parser with
	// a per-repository secret key used to verify the webhook
	// payload signature for authenticity.
	fn := func(webhook scm.Webhook) (string, error) {
		if webhook == nil {
			// HACK(bradrydzewski) if the incoming webhook is nil
			// we assume it is an unknown event or action. A more
			// permanent fix is to update go-scm to return an
			// scm.ErrUnknownAction error.
			return "", scm.ErrUnknownEvent
		}
		repo := webhook.Repository()
		slug := scm.Join(repo.Namespace, repo.Name)
		secret := secretFunc(slug)
		if secret == "" {
			return secret, errors.New("Cannot find repository")
		}
		return secret, nil
	}

	payload, err := p.client.Webhooks.Parse(req, fn)
	if err == scm.ErrUnknownEvent {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	var repo *core.Repository
	var hook *core.Hook

	switch v := payload.(type) {
	case *scm.PushHook:
		// github sends push hooks when tags and branches are
		// deleted. These hooks should be ignored.
		if v.Commit.Sha == emptyCommit {
			return nil, nil, nil
		}
		// github sends push hooks when tags are created. The
		// push hook contains more information than the tag hook,
		// so we choose to use the push hook for tags.
		if strings.HasPrefix(v.Ref, "refs/tags/") {
			hook = &core.Hook{
				Trigger:      core.TriggerHook, // core.TriggerHook
				Event:        core.EventTag,
				Action:       core.ActionCreate,
				Link:         v.Commit.Link,
				Timestamp:    v.Commit.Author.Date.Unix(),
				Message:      v.Commit.Message,
				Before:       v.Before,
				After:        v.Commit.Sha,
				Source:       scm.TrimRef(v.BaseRef),
				Target:       scm.TrimRef(v.BaseRef),
				Ref:          v.Ref,
				Author:       v.Commit.Author.Login,
				AuthorName:   v.Commit.Author.Name,
				AuthorEmail:  v.Commit.Author.Email,
				AuthorAvatar: v.Commit.Author.Avatar,
				Sender:       v.Sender.Login,
			}
		} else {
			hook = &core.Hook{
				Trigger:      core.TriggerHook, //core.TriggerHook,
				Event:        core.EventPush,
				Link:         v.Commit.Link,
				Timestamp:    v.Commit.Author.Date.Unix(),
				Message:      v.Commit.Message,
				Before:       v.Before,
				After:        v.Commit.Sha,
				Ref:          v.Ref,
				Source:       strings.TrimPrefix(v.Ref, "refs/heads/"),
				Target:       strings.TrimPrefix(v.Ref, "refs/heads/"),
				Author:       v.Commit.Author.Login,
				AuthorName:   v.Commit.Author.Name,
				AuthorEmail:  v.Commit.Author.Email,
				AuthorAvatar: v.Commit.Author.Avatar,
				Sender:       v.Sender.Login,
			}
		}
		repo = &core.Repository{
			UID:       v.Repo.ID,
			Namespace: v.Repo.Namespace,
			Name:      v.Repo.Name,
			Slug:      scm.Join(v.Repo.Namespace, v.Repo.Name),
			Link:      v.Repo.Link,
			Branch:    v.Repo.Branch,
			Private:   v.Repo.Private,
			HTTPURL:   v.Repo.Clone,
			SSHURL:    v.Repo.CloneSSH,
		}
		// gogs and gitea do not include the author avatar in
		// the webhook, but they do include the sender avatar.
		// use the sender avatar when necessary.
		if hook.AuthorAvatar == "" {
			hook.AuthorAvatar = v.Sender.Avatar
		}
		return hook, repo, nil
	case *scm.TagHook:
		if v.Action != scm.ActionCreate {
			return nil, nil, nil
		}
		// when a tag is created github sends both a push hook
		// and a tag create hook. The push hook contains more
		// information, so we choose to use the push hook and
		// ignore the native tag hook.
		if p.client.Driver == scm.DriverGithub ||
			p.client.Driver == scm.DriverGitea ||
			p.client.Driver == scm.DriverGitlab {
			return nil, nil, nil
		}

		// the tag hook does not include the commit link, message
		// or timestamp. In some cases it does not event include
		// the sha (gogs). Note that we may need to fetch additional
		// details to augment the webhook.
		hook = &core.Hook{
			Trigger:      core.TriggerHook, // core.TriggerHook,
			Event:        core.EventTag,
			Action:       core.ActionCreate,
			Link:         "",
			Timestamp:    0,
			Message:      "",
			After:        v.Ref.Sha,
			Ref:          v.Ref.Name,
			Source:       v.Ref.Name,
			Target:       v.Ref.Name,
			Author:       v.Sender.Login,
			AuthorName:   v.Sender.Name,
			AuthorEmail:  v.Sender.Email,
			AuthorAvatar: v.Sender.Avatar,
			Sender:       v.Sender.Login,
		}
		repo = &core.Repository{
			UID:       v.Repo.ID,
			Namespace: v.Repo.Namespace,
			Name:      v.Repo.Name,
			Slug:      scm.Join(v.Repo.Namespace, v.Repo.Name),
			Link:      v.Repo.Link,
			Branch:    v.Repo.Branch,
			Private:   v.Repo.Private,
			HTTPURL:   v.Repo.Clone,
			SSHURL:    v.Repo.CloneSSH,
		}
		// TODO(bradrydzewski) can we use scm.ExpandRef here?
		if !strings.HasPrefix(hook.Ref, "refs/tags/") {
			hook.Ref = fmt.Sprintf("refs/tags/%s", hook.Ref)
		}
		if hook.AuthorAvatar == "" {
			hook.AuthorAvatar = v.Sender.Avatar
		}
		return hook, repo, nil
	case *scm.PullRequestHook:
		// TODO(bradrydzewski) cleanup the pr close hook code.
		if v.Action == scm.ActionClose {
			return &core.Hook{
					Trigger: core.TriggerHook,
					Event:   core.EventPullRequest,
					Action:  core.ActionClose,
					After:   v.PullRequest.Sha,
					Ref:     v.PullRequest.Ref,
				}, &core.Repository{
					UID:       v.Repo.ID,
					Namespace: v.Repo.Namespace,
					Name:      v.Repo.Name,
					Slug:      scm.Join(v.Repo.Namespace, v.Repo.Name),
				}, nil
		}

		switch v.Action {
		case scm.ActionOpen, scm.ActionClose, scm.ActionSync:
			// Valid, continue
		default:
			return nil, nil, nil
		}

		// Pull Requests are not supported for Bitbucket due
		// to lack of refs (e.g. refs/pull-requests/42/from).
		// Please contact Bitbucket Support if you would like to
		// see this feature enabled:
		// https://bitbucket.org/site/master/issues/5814/repository-refs-for-pull-requests
		if p.client.Driver == scm.DriverBitbucket {
			return nil, nil, nil
		}

		hook = &core.Hook{
			Trigger:      core.TriggerHook, // core.TriggerHook,
			Event:        core.EventPullRequest,
			Action:       v.Action.String(),
			Link:         v.PullRequest.Link,
			Timestamp:    v.PullRequest.Created.Unix(),
			Title:        v.PullRequest.Title,
			Message:      v.PullRequest.Body,
			Before:       v.PullRequest.Base.Sha,
			After:        v.PullRequest.Sha,
			Ref:          v.PullRequest.Ref,
			Fork:         v.PullRequest.Fork,
			Source:       v.PullRequest.Source,
			Target:       v.PullRequest.Target,
			Author:       v.PullRequest.Author.Login,
			AuthorName:   v.PullRequest.Author.Name,
			AuthorEmail:  v.PullRequest.Author.Email,
			AuthorAvatar: v.PullRequest.Author.Avatar,
			Sender:       v.Sender.Login,
		}
		// HACK this is a workaround for github. The pull
		// request title is populated, but not the message.
		if hook.Message == "" {
			hook.Message = hook.Title
		}
		repo = &core.Repository{
			UID:       v.Repo.ID,
			Namespace: v.Repo.Namespace,
			Name:      v.Repo.Name,
			Slug:      scm.Join(v.Repo.Namespace, v.Repo.Name),
			Link:      v.Repo.Link,
			Branch:    v.Repo.Branch,
			Private:   v.Repo.Private,
			HTTPURL:   v.Repo.Clone,
			SSHURL:    v.Repo.CloneSSH,
		}
		if hook.AuthorAvatar == "" {
			hook.AuthorAvatar = v.Sender.Avatar
		}
		return hook, repo, nil
	case *scm.BranchHook:

		// TODO(bradrydzewski) cleanup the branch hook code.
		if v.Action == scm.ActionDelete {
			return &core.Hook{
					Trigger: core.TriggerHook,
					Event:   core.EventPush,
					After:   v.Ref.Sha,
					Action:  core.ActionDelete,
					Target:  scm.TrimRef(v.Ref.Name),
				}, &core.Repository{
					UID:       v.Repo.ID,
					Namespace: v.Repo.Namespace,
					Name:      v.Repo.Name,
					Slug:      scm.Join(v.Repo.Namespace, v.Repo.Name),
				}, nil
		}

		if v.Action != scm.ActionCreate {
			return nil, nil, nil
		}
		if p.client.Driver != scm.DriverStash {
			return nil, nil, nil
		}
		hook = &core.Hook{
			Trigger:      core.TriggerHook, // core.TriggerHook,
			Event:        core.EventPush,
			Link:         "",
			Timestamp:    0,
			Message:      "",
			After:        v.Ref.Sha,
			Ref:          v.Ref.Name,
			Source:       v.Ref.Name,
			Target:       v.Ref.Name,
			Author:       v.Sender.Login,
			AuthorName:   v.Sender.Name,
			AuthorEmail:  v.Sender.Email,
			AuthorAvatar: v.Sender.Avatar,
			Sender:       v.Sender.Login,
		}
		repo = &core.Repository{
			UID:       v.Repo.ID,
			Namespace: v.Repo.Namespace,
			Name:      v.Repo.Name,
			Slug:      scm.Join(v.Repo.Namespace, v.Repo.Name),
			Link:      v.Repo.Link,
			Branch:    v.Repo.Branch,
			Private:   v.Repo.Private,
			HTTPURL:   v.Repo.Clone,
			SSHURL:    v.Repo.CloneSSH,
		}
		return hook, repo, nil
	case *scm.DeployHook:
		hook = &core.Hook{
			Trigger:      core.TriggerHook,
			Event:        core.EventPromote,
			Link:         v.TargetURL,
			Timestamp:    time.Now().Unix(),
			Message:      v.Desc,
			After:        v.Ref.Sha,
			Ref:          v.Ref.Path,
			Source:       v.Ref.Name,
			Target:       v.Ref.Name,
			Author:       v.Sender.Login,
			AuthorName:   v.Sender.Name,
			AuthorEmail:  v.Sender.Email,
			AuthorAvatar: v.Sender.Avatar,
			Sender:       v.Sender.Login,
			Deployment:   v.Target,
			DeploymentID: v.Number,
			Params:       toMap(v.Data),
		}
		repo = &core.Repository{
			UID:       v.Repo.ID,
			Namespace: v.Repo.Namespace,
			Name:      v.Repo.Name,
			Slug:      scm.Join(v.Repo.Namespace, v.Repo.Name),
			Link:      v.Repo.Link,
			Branch:    v.Repo.Branch,
			Private:   v.Repo.Private,
			HTTPURL:   v.Repo.Clone,
			SSHURL:    v.Repo.CloneSSH,
		}
		return hook, repo, nil
	default:
		return nil, nil, nil
	}
}

func toMap(src interface{}) map[string]string {
	set, ok := src.(map[string]interface{})
	if !ok {
		return nil
	}
	dst := map[string]string{}
	for k, v := range set {
		dst[k] = fmt.Sprint(v)
	}
	return dst
}
