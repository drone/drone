// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package builds

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/request"
	"github.com/drone/drone/mock"

	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

func TestCreate(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockCommit := &core.Commit{
		Sha:     "cce10d5c4760d1d6ede99db850ab7e77efe15579",
		Ref:     "refs/heads/master",
		Message: "updated README.md",
		Link:    "https://github.com/octocatl/hello-world/commit/cce10d5c4760d1d6ede99db850ab7e77efe15579",
		Author: &core.Committer{
			Name:   "The Octocat",
			Email:  "octocat@github.com",
			Login:  "octocat",
			Avatar: "https://github.com/octocat.png",
		},
	}

	checkBuild := func(_ context.Context, _ *core.Repository, hook *core.Hook) error {
		if got, want := hook.Trigger, mockUser.Login; got != want {
			t.Errorf("Want hook Trigger By %s, got %s", want, got)
		}
		if got, want := hook.Event, core.EventCustom; got != want {
			t.Errorf("Want hook Event %s, got %s", want, got)
		}
		if got, want := hook.Link, mockCommit.Link; got != want {
			t.Errorf("Want hook Link %s, got %s", want, got)
		}
		if got, want := hook.Message, mockCommit.Message; got != want {
			t.Errorf("Want hook Message %s, got %s", want, got)
		}
		if got, want := hook.Before, mockCommit.Sha; got != want {
			t.Errorf("Want hook Before %s, got %s", want, got)
		}
		if got, want := hook.After, mockCommit.Sha; got != want {
			t.Errorf("Want hook After %s, got %s", want, got)
		}
		if got, want := hook.Ref, mockCommit.Ref; got != want {
			t.Errorf("Want hook Ref %s, got %s", want, got)
		}
		if got, want := hook.Source, "master"; got != want {
			t.Errorf("Want hook Source %s, got %s", want, got)
		}
		if got, want := hook.Target, "master"; got != want {
			t.Errorf("Want hook Target %s, got %s", want, got)
		}
		if got, want := hook.Author, mockCommit.Author.Login; got != want {
			t.Errorf("Want hook Author %s, got %s", want, got)
		}
		if got, want := hook.AuthorName, mockCommit.Author.Name; got != want {
			t.Errorf("Want hook AuthorName %s, got %s", want, got)
		}
		if got, want := hook.AuthorEmail, mockCommit.Author.Email; got != want {
			t.Errorf("Want hook AuthorEmail %s, got %s", want, got)
		}
		if got, want := hook.AuthorAvatar, mockCommit.Author.Avatar; got != want {
			t.Errorf("Want hook AuthorAvatar %s, got %s", want, got)
		}
		if got, want := hook.Sender, mockUser.Login; got != want {
			t.Errorf("Want hook Sender %s, got %s", want, got)
		}
		return nil
	}

	users := mock.NewMockUserStore(controller)
	users.EXPECT().Find(gomock.Any(), mockRepo.UserID).Return(mockUser, nil)

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	commits := mock.NewMockCommitService(controller)
	commits.EXPECT().Find(gomock.Any(), mockUser, mockRepo.Slug, mockCommit.Sha).Return(mockCommit, nil)

	triggerer := mock.NewMockTriggerer(controller)
	triggerer.EXPECT().Trigger(gomock.Any(), mockRepo, gomock.Any()).Return(mockBuild, nil).Do(checkBuild)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")

	params := &url.Values{}
	params.Set("branch", "master")
	params.Set("commit", mockCommit.Sha)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/?"+params.Encode(), nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleCreate(users, repos, commits, triggerer)(w, r)
	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(core.Build), mockBuild
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

func TestCreate_FromHead(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockCommit := &core.Commit{
		Sha:     "cce10d5c4760d1d6ede99db850ab7e77efe15579",
		Ref:     "refs/heads/master",
		Message: "updated README.md",
		Link:    "https://github.com/octocatl/hello-world/commit/cce10d5c4760d1d6ede99db850ab7e77efe15579",
		Author: &core.Committer{
			Name:   "The Octocat",
			Email:  "octocat@github.com",
			Login:  "octocat",
			Avatar: "https://github.com/octocat.png",
		},
	}

	users := mock.NewMockUserStore(controller)
	users.EXPECT().Find(gomock.Any(), mockRepo.UserID).Return(mockUser, nil)

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	commits := mock.NewMockCommitService(controller)
	commits.EXPECT().FindRef(gomock.Any(), mockUser, mockRepo.Slug, mockCommit.Ref).Return(mockCommit, nil)

	triggerer := mock.NewMockTriggerer(controller)
	triggerer.EXPECT().Trigger(gomock.Any(), mockRepo, gomock.Any()).Return(mockBuild, nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleCreate(users, repos, commits, triggerer)(w, r)
	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(core.Build), mockBuild
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

func TestCreate_FromHead_JSON_body(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockCommit := &core.Commit{
		Sha:     "cce10d5c4760d1d6ede99db850ab7e77efe15579",
		Ref:     "refs/heads/master",
		Message: "updated README.md",
		Link:    "https://github.com/octocatl/hello-world/commit/cce10d5c4760d1d6ede99db850ab7e77efe15579",
		Author: &core.Committer{
			Name:   "The Octocat",
			Email:  "octocat@github.com",
			Login:  "octocat",
			Avatar: "https://github.com/octocat.png",
		},
	}

	users := mock.NewMockUserStore(controller)
	users.EXPECT().Find(gomock.Any(), mockRepo.UserID).Return(mockUser, nil)

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), gomock.Any(), mockRepo.Name).Return(mockRepo, nil)

	commits := mock.NewMockCommitService(controller)
	commits.EXPECT().FindRef(gomock.Any(), mockUser, mockRepo.Slug, mockCommit.Ref).Return(mockCommit, nil)

	triggerer := mock.NewMockTriggerer(controller)
	triggerer.EXPECT().Trigger(gomock.Any(), mockRepo, gomock.Any()).Return(mockBuild, nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")

	w := httptest.NewRecorder()
	testPostBody := map[string]interface{}{
		"test": "test",
	}
	body, _ := json.Marshal(testPostBody)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(
		context.WithValue(request.WithUser(r.Context(), mockUser), chi.RouteCtxKey, c),
	)

	HandleCreate(users, repos, commits, triggerer)(w, r)
	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(core.Build), mockBuild
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}
