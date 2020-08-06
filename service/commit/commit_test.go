// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package commit

import (
	"context"
	"testing"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"
	"github.com/drone/drone/mock/mockscm"
	"github.com/drone/go-scm/scm"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

var noContext = context.Background()

func TestFind(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}
	mockCommit := &scm.Commit{
		Sha:     "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
		Message: "Merge pull request #6 from Spaceghost/patch-1\n\nNew line at end of file.",
		Author: scm.Signature{
			Name:   "The Octocat",
			Email:  "octocat@nowhere.com",
			Date:   time.Unix(1532303087, 0),
			Login:  "octocat",
			Avatar: "https://avatars3.githubusercontent.com/u/583231?v=4",
		},
		Committer: scm.Signature{
			Name:   "The Octocat",
			Email:  "octocat@nowhere.com",
			Date:   time.Unix(1532303087, 0),
			Login:  "octocat",
			Avatar: "https://avatars3.githubusercontent.com/u/583231?v=4",
		},
		Link: "https://github.com/octocat/Hello-World/commit/7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
	}

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false).Return(nil)

	mockGit := mockscm.NewMockGitService(controller)
	mockGit.EXPECT().FindCommit(gomock.Any(), "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa").Return(mockCommit, nil, nil)

	client := new(scm.Client)
	client.Git = mockGit

	want := &core.Commit{
		Sha:     "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
		Ref:     "",
		Message: "Merge pull request #6 from Spaceghost/patch-1\n\nNew line at end of file.",
		Author: &core.Committer{
			Name:   "The Octocat",
			Email:  "octocat@nowhere.com",
			Date:   1532303087,
			Login:  "octocat",
			Avatar: "https://avatars3.githubusercontent.com/u/583231?v=4",
		},
		Committer: &core.Committer{
			Name:   "The Octocat",
			Email:  "octocat@nowhere.com",
			Date:   1532303087,
			Login:  "octocat",
			Avatar: "https://avatars3.githubusercontent.com/u/583231?v=4",
		},
		Link: "https://github.com/octocat/Hello-World/commit/7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
	}

	service := New(client, mockRenewer)
	got, err := service.Find(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa")
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestFind_Err(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false).Return(nil)

	mockGit := mockscm.NewMockGitService(controller)
	mockGit.EXPECT().FindCommit(gomock.Any(), "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa").Return(nil, nil, scm.ErrNotFound)

	client := new(scm.Client)
	client.Git = mockGit

	service := New(client, mockRenewer)
	_, err := service.Find(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa")
	if err != scm.ErrNotFound {
		t.Errorf("Want not found error, got %v", err)
	}
}

func TestFind_ErrRenew(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false).Return(scm.ErrNotAuthorized)

	service := New(nil, mockRenewer)
	_, err := service.Find(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa")
	if err != scm.ErrNotAuthorized {
		t.Errorf("Want not authorized error, got %v", err)
	}
}

func TestFindRef(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}
	mockCommit := &scm.Commit{
		Sha:     "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
		Message: "Merge pull request #6 from Spaceghost/patch-1\n\nNew line at end of file.",
		Author: scm.Signature{
			Name:   "The Octocat",
			Email:  "octocat@nowhere.com",
			Date:   time.Unix(1532303087, 0),
			Login:  "octocat",
			Avatar: "https://avatars3.githubusercontent.com/u/583231?v=4",
		},
		Committer: scm.Signature{
			Name:   "The Octocat",
			Email:  "octocat@nowhere.com",
			Date:   time.Unix(1532303087, 0),
			Login:  "octocat",
			Avatar: "https://avatars3.githubusercontent.com/u/583231?v=4",
		},
		Link: "https://github.com/octocat/Hello-World/commit/7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
	}

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false).Return(nil)

	mockGit := mockscm.NewMockGitService(controller)
	mockGit.EXPECT().FindCommit(gomock.Any(), "octocat/hello-world", "master").Return(mockCommit, nil, nil)

	client := new(scm.Client)
	client.Git = mockGit

	want := &core.Commit{
		Sha:     "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
		Ref:     "master",
		Message: "Merge pull request #6 from Spaceghost/patch-1\n\nNew line at end of file.",
		Author: &core.Committer{
			Name:   "The Octocat",
			Email:  "octocat@nowhere.com",
			Date:   1532303087,
			Login:  "octocat",
			Avatar: "https://avatars3.githubusercontent.com/u/583231?v=4",
		},
		Committer: &core.Committer{
			Name:   "The Octocat",
			Email:  "octocat@nowhere.com",
			Date:   1532303087,
			Login:  "octocat",
			Avatar: "https://avatars3.githubusercontent.com/u/583231?v=4",
		},
		Link: "https://github.com/octocat/Hello-World/commit/7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
	}

	service := New(client, mockRenewer)
	got, err := service.FindRef(noContext, mockUser, "octocat/hello-world", "master")
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestFindRef_Err(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false).Return(nil)

	mockGit := mockscm.NewMockGitService(controller)
	mockGit.EXPECT().FindCommit(gomock.Any(), "octocat/hello-world", "master").Return(nil, nil, scm.ErrNotFound)

	client := new(scm.Client)
	client.Git = mockGit

	service := New(client, mockRenewer)
	_, err := service.FindRef(noContext, mockUser, "octocat/hello-world", "master")
	if err != scm.ErrNotFound {
		t.Errorf("Want not found error, got %v", err)
	}
}

func TestFindRef_ErrRenew(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false).Return(scm.ErrNotAuthorized)

	service := New(nil, mockRenewer)
	_, err := service.FindRef(noContext, mockUser, "octocat/hello-world", "master")
	if err != scm.ErrNotAuthorized {
		t.Errorf("Want not authorized error, got %v", err)
	}
}

func TestListChanges(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}
	mockChanges := []*scm.Change{
		{
			Path:    "file1",
			Added:   false,
			Deleted: false,
			Renamed: false,
		},
		{
			Path:    "file2",
			Added:   true,
			Deleted: false,
			Renamed: false,
		},
		{
			Path:    "file2",
			Added:   false,
			Deleted: true,
			Renamed: false,
		},
		{
			Path:    "file3",
			Added:   false,
			Deleted: false,
			Renamed: true,
		},
	}

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false).Return(nil)

	mockGit := mockscm.NewMockGitService(controller)
	mockGit.EXPECT().ListChanges(gomock.Any(), "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", gomock.Any()).Return(mockChanges, nil, nil)

	client := new(scm.Client)
	client.Git = mockGit

	want := []*core.Change{
		{
			Path:    "file1",
			Added:   false,
			Deleted: false,
			Renamed: false,
		},
		{
			Path:    "file2",
			Added:   true,
			Deleted: false,
			Renamed: false,
		},
		{
			Path:    "file2",
			Added:   false,
			Deleted: true,
			Renamed: false,
		},
		{
			Path:    "file3",
			Added:   false,
			Deleted: false,
			Renamed: true,
		},
	}

	service := New(client, mockRenewer)
	got, err := service.ListChanges(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", "master")
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestListChanges_Err(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false).Return(nil)

	mockGit := mockscm.NewMockGitService(controller)
	mockGit.EXPECT().ListChanges(gomock.Any(), "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", gomock.Any()).Return(nil, nil, scm.ErrNotFound)

	client := new(scm.Client)
	client.Git = mockGit

	service := New(client, mockRenewer)
	_, err := service.ListChanges(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", "master")
	if err != scm.ErrNotFound {
		t.Errorf("Want not found error, got %v", err)
	}
}

func TestListChanges_ErrRenew(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUser := &core.User{}

	mockRenewer := mock.NewMockRenewer(controller)
	mockRenewer.EXPECT().Renew(gomock.Any(), mockUser, false).Return(scm.ErrNotAuthorized)

	service := New(nil, mockRenewer)
	_, err := service.ListChanges(noContext, mockUser, "octocat/hello-world", "a6586b3db244fb6b1198f2b25c213ded5b44f9fa", "master")
	if err != scm.ErrNotAuthorized {
		t.Errorf("Want not authorized error, got %v", err)
	}
}
