// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package trigger

import (
	"context"
	"database/sql"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"
	"github.com/sirupsen/logrus"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var noContext = context.Background()

func init() {
	logrus.SetOutput(ioutil.Discard)
}

func TestTrigger(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	checkBuild := func(_ context.Context, build *core.Build, stages []*core.Stage) {
		if diff := cmp.Diff(build, dummyBuild, ignoreBuildFields); diff != "" {
			t.Errorf(diff)
		}
		if diff := cmp.Diff(stages, dummyStages, ignoreStageFields); diff != "" {
			t.Errorf(diff)
		}
	}

	checkStatus := func(_ context.Context, _ *core.User, req *core.StatusInput) error {
		if diff := cmp.Diff(req.Build, dummyBuild, ignoreBuildFields); diff != "" {
			t.Errorf(diff)
		}
		if diff := cmp.Diff(req.Repo, dummyRepo, ignoreStageFields); diff != "" {
			t.Errorf(diff)
		}
		return nil
	}

	mockUsers := mock.NewMockUserStore(controller)
	mockUsers.EXPECT().Find(gomock.Any(), dummyRepo.UserID).Return(dummyUser, nil)

	mockRepos := mock.NewMockRepositoryStore(controller)
	mockRepos.EXPECT().Increment(gomock.Any(), dummyRepo).Return(dummyRepo, nil)

	mockConfigService := mock.NewMockConfigService(controller)
	mockConfigService.EXPECT().Find(gomock.Any(), gomock.Any()).Return(dummyYaml, nil)

	mockConvertService := mock.NewMockConvertService(controller)
	mockConvertService.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(dummyYaml, nil)

	mockValidateService := mock.NewMockValidateService(controller)
	mockValidateService.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)

	mockStatus := mock.NewMockStatusService(controller)
	mockStatus.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Do(checkStatus)

	mockQueue := mock.NewMockScheduler(controller)
	mockQueue.EXPECT().Schedule(gomock.Any(), gomock.Any()).Return(nil)

	mockBuilds := mock.NewMockBuildStore(controller)
	mockBuilds.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Do(checkBuild).Return(nil)

	mockWebhooks := mock.NewMockWebhookSender(controller)
	mockWebhooks.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)

	triggerer := New(
		nil,
		mockConfigService,
		mockConvertService,
		nil,
		mockStatus,
		mockBuilds,
		mockQueue,
		mockRepos,
		mockUsers,
		mockValidateService,
		mockWebhooks,
	)

	build, err := triggerer.Trigger(noContext, dummyRepo, dummyHook)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(500 * time.Millisecond)
	if diff := cmp.Diff(build, dummyBuild, ignoreBuildFields); diff != "" {
		t.Errorf(diff)
	}
}

// this test verifies that hook is ignored if the commit
// message includes the [CI SKIP] keyword.
func TestTrigger_SkipCI(t *testing.T) {
	triggerer := New(
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	dummyHookSkip := *dummyHook
	dummyHookSkip.Message = "foo [CI SKIP] bar"
	triggerer.Trigger(noContext, dummyRepo, &dummyHookSkip)
}

// this test verifies that if the system cannot determine
// the repository owner, the function must exit with an error.
// The owner is required because we need an oauth token
// when fetching the configuration file.
func TestTrigger_NoOwner(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUsers := mock.NewMockUserStore(controller)
	mockUsers.EXPECT().Find(noContext, dummyRepo.UserID).Return(nil, sql.ErrNoRows)

	triggerer := New(
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		mockUsers,
		nil,
		nil,
	)

	_, err := triggerer.Trigger(noContext, dummyRepo, dummyHook)
	if err != sql.ErrNoRows {
		t.Errorf("Expect error when yaml not found")
	}
}

// this test verifies that if the system cannot fetch the yaml
// configuration file, the function must exit with an error.
func TestTrigger_MissingYaml(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	ignoreBuildFields := cmpopts.IgnoreFields(core.Build{},
		"Created", "Updated", "Finished", "Started")

	checkBuild := func(_ context.Context, build *core.Build, stages []*core.Stage) {
		// copy dumy build
		dummyBuildWithErr := *dummyBuild
		dummyBuildWithErr.Status = core.StatusError
		dummyBuildWithErr.Error = "EOF"

		if diff := cmp.Diff(build, &dummyBuildWithErr, ignoreBuildFields); diff != "" {
			t.Errorf(diff)
		}
	}

	mockUsers := mock.NewMockUserStore(controller)
	mockUsers.EXPECT().Find(noContext, dummyRepo.UserID).Return(dummyUser, nil)

	mockConfigService := mock.NewMockConfigService(controller)
	mockConfigService.EXPECT().Find(gomock.Any(), gomock.Any()).Return(nil, io.EOF)

	mockRepos := mock.NewMockRepositoryStore(controller)
	mockRepos.EXPECT().Increment(gomock.Any(), dummyRepo).Return(dummyRepo, nil)

	mockBuilds := mock.NewMockBuildStore(controller)
	mockBuilds.EXPECT().Create(gomock.Any(), gomock.Any(), nil).Do(checkBuild).Return(nil)

	triggerer := New(
		nil,
		mockConfigService,
		nil,
		nil,
		nil,
		mockBuilds,
		nil,
		mockRepos,
		mockUsers,
		nil,
		nil,
	)

	_, err := triggerer.Trigger(noContext, dummyRepo, dummyHook)
	if err != nil {
		t.Errorf("Yaml not found should return no error as config is ran async")
	}
	// wait for goroutine to complete
	time.Sleep(500 * time.Millisecond)
}

// this test verifies that if the system cannot parse the yaml
// configuration file, the function exits with no error and the build
// is created with an error.
func TestTrigger_ErrorYaml(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	ignoreBuildFields := cmpopts.IgnoreFields(core.Build{},
		"Created", "Updated", "Finished", "Started")

	checkBuild := func(_ context.Context, build *core.Build, stages []*core.Stage) {
		// copy dumy build
		dummyBuildWithErr := *dummyBuild
		dummyBuildWithErr.Status = core.StatusError
		dummyBuildWithErr.Error = "yaml: found unknown directive name"

		if diff := cmp.Diff(build, &dummyBuildWithErr, ignoreBuildFields); diff != "" {
			t.Errorf(diff)
		}
	}

	mockUsers := mock.NewMockUserStore(controller)
	mockUsers.EXPECT().Find(noContext, dummyRepo.UserID).Return(dummyUser, nil)

	mockConfigService := mock.NewMockConfigService(controller)
	mockConfigService.EXPECT().Find(gomock.Any(), gomock.Any()).Return(dummyYamlInvalid, nil)

	mockConvertService := mock.NewMockConvertService(controller)
	mockConvertService.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(dummyYamlInvalid, nil)

	mockRepos := mock.NewMockRepositoryStore(controller)
	mockRepos.EXPECT().Increment(gomock.Any(), dummyRepo).Return(dummyRepo, nil)

	mockBuilds := mock.NewMockBuildStore(controller)
	mockBuilds.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Do(checkBuild).Return(nil)

	triggerer := New(
		nil,
		mockConfigService,
		mockConvertService,
		nil,
		nil,
		mockBuilds,
		nil,
		mockRepos,
		mockUsers,
		nil,
		nil,
	)

	_, err := triggerer.Trigger(noContext, dummyRepo, dummyHook)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(500 * time.Millisecond)
}

// this test verifies that no build should be scheduled if the
// hook branch does not match the branches defined in the yaml.
func TestTrigger_SkipBranch(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUsers := mock.NewMockUserStore(controller)
	mockUsers.EXPECT().Find(noContext, dummyRepo.UserID).Return(dummyUser, nil)

	mockRepos := mock.NewMockRepositoryStore(controller)
	mockRepos.EXPECT().Increment(gomock.Any(), dummyRepo).Return(dummyRepo, nil)

	mockConfigService := mock.NewMockConfigService(controller)
	mockConfigService.EXPECT().Find(gomock.Any(), gomock.Any()).Return(dummyYamlSkipBranch, nil)

	mockConvertService := mock.NewMockConvertService(controller)
	mockConvertService.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(dummyYamlSkipBranch, nil)

	mockValidateService := mock.NewMockValidateService(controller)
	mockValidateService.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)

	triggerer := New(
		nil,
		mockConfigService,
		mockConvertService,
		nil,
		nil,
		nil,
		nil,
		mockRepos,
		mockUsers,
		mockValidateService,
		nil,
	)

	_, err := triggerer.Trigger(noContext, dummyRepo, dummyHook)
	if err != nil {
		t.Errorf("Expect build silently skipped if branch does not match")
	}
	time.Sleep(500 * time.Millisecond)
}

// this test verifies that no build should be scheduled if the
// hook event does not match the events defined in the yaml.
func TestTrigger_SkipEvent(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUsers := mock.NewMockUserStore(controller)
	mockUsers.EXPECT().Find(noContext, dummyRepo.UserID).Return(dummyUser, nil)

	mockRepos := mock.NewMockRepositoryStore(controller)
	mockRepos.EXPECT().Increment(gomock.Any(), dummyRepo).Return(dummyRepo, nil)

	mockConfigService := mock.NewMockConfigService(controller)
	mockConfigService.EXPECT().Find(gomock.Any(), gomock.Any()).Return(dummyYamlSkipEvent, nil)

	mockConvertService := mock.NewMockConvertService(controller)
	mockConvertService.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(dummyYamlSkipEvent, nil)

	mockValidateService := mock.NewMockValidateService(controller)
	mockValidateService.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)

	triggerer := New(
		nil,
		mockConfigService,
		mockConvertService,
		nil,
		nil,
		nil,
		nil,
		mockRepos,
		mockUsers,
		mockValidateService,
		nil,
	)

	_, err := triggerer.Trigger(noContext, dummyRepo, dummyHook)
	if err != nil {
		t.Errorf("Expect build silently skipped if event does not match")
	}
	time.Sleep(500 * time.Millisecond)
}

// this test verifies that no build should be scheduled if the
// hook action does not match the actions defined in the yaml.
func TestTrigger_SkipAction(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUsers := mock.NewMockUserStore(controller)
	mockUsers.EXPECT().Find(noContext, dummyRepo.UserID).Return(dummyUser, nil)

	mockRepos := mock.NewMockRepositoryStore(controller)
	mockRepos.EXPECT().Increment(gomock.Any(), dummyRepo).Return(dummyRepo, nil)

	mockConfigService := mock.NewMockConfigService(controller)
	mockConfigService.EXPECT().Find(gomock.Any(), gomock.Any()).Return(dummyYamlSkipAction, nil)

	mockConvertService := mock.NewMockConvertService(controller)
	mockConvertService.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(dummyYamlSkipAction, nil)

	mockValidateService := mock.NewMockValidateService(controller)
	mockValidateService.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)

	triggerer := New(
		nil,
		mockConfigService,
		mockConvertService,
		nil,
		nil,
		nil,
		nil,
		mockRepos,
		mockUsers,
		mockValidateService,
		nil,
	)

	_, err := triggerer.Trigger(noContext, dummyRepo, dummyHook)
	if err != nil {
		t.Errorf("Expect build silently skipped if action does not match")
	}
	time.Sleep(500 * time.Millisecond)
}

// this test verifies that if the system cannot increment the
// build number, the function must exit with error and must not
// schedule a new build.
func TestTrigger_ErrorIncrement(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockUsers := mock.NewMockUserStore(controller)
	mockUsers.EXPECT().Find(noContext, dummyRepo.UserID).Return(dummyUser, nil)

	mockRepos := mock.NewMockRepositoryStore(controller)
	mockRepos.EXPECT().Increment(gomock.Any(), dummyRepo).Return(nil, sql.ErrNoRows)

	triggerer := New(
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		mockRepos,
		mockUsers,
		nil,
		nil,
	)

	_, err := triggerer.Trigger(noContext, dummyRepo, dummyHook)
	if err != sql.ErrNoRows {
		t.Errorf("Expect error when unable to increment build sequence")
	}
}

func TestTrigger_ErrorCreate(t *testing.T) {
	t.Skip()
	// 	controller := gomock.NewController(t)
	// 	defer controller.Finish()

	// 	mockUsers := mock.NewMockUserStore(controller)
	// 	mockUsers.EXPECT().Find(noContext, dummyRepo.UserID).Return(dummyUser, nil)

	// 	mockTriggers := mock.NewMockTriggerStore(controller)
	// 	mockTriggers.EXPECT().List(noContext, dummyRepo.ID).Return([]*core.Trigger{dummyTrigger}, nil)

	// 	mockRepos := mock.NewMockRepositoryStore(controller)
	// 	mockRepos.EXPECT().Increment(gomock.Any(), dummyRepo).Return(dummyRepo, nil)

	// 	mockContents := mock.NewMockContentService(controller)
	// 	mockContents.EXPECT().Find(gomock.Any(), dummyRepo.Slug, dummyTrigger.Path, dummyHook.After).Return(dummyYaml, nil, nil)
	// 	mockContents.EXPECT().Find(gomock.Any(), dummyRepo.Slug, dummySignature.Path, dummyHook.After).Return(dummySignature, nil, nil)

	// 	mockClient := new(scm.Client)
	// 	mockClient.Contents = mockContents

	// 	mockBuilds := mock.NewMockBuildStore(controller)
	// 	mockBuilds.EXPECT().Create(gomock.Any(), gomock.Any()).Return(sql.ErrNoRows)

	// 	triggerer := New(
	// 		mockClient,
	// 		mockBuilds,
	// 		nil,
	// 		mockRepos,
	// 		mockTriggers,
	// 		mockUsers,
	// 	)

	// 	builds, err := triggerer.Trigger(noContext, dummyRepo, dummyHook)
	// 	if err != sql.ErrNoRows {
	// 		t.Error("Expect error when persisting the build fails")
	// 	}
	// 	if got, want := len(builds), 0; got != want {
	// 		t.Errorf("Got build count %d, want %d", got, want)
	// 	}
}

func TestTrigger_ErrorEnqueue(t *testing.T) {
	t.Skip()
	// 	controller := gomock.NewController(t)
	// 	defer controller.Finish()

	// 	mockUsers := mock.NewMockUserStore(controller)
	// 	mockUsers.EXPECT().Find(noContext, dummyRepo.UserID).Return(dummyUser, nil)

	// 	mockTriggers := mock.NewMockTriggerStore(controller)
	// 	mockTriggers.EXPECT().List(noContext, dummyRepo.ID).Return([]*core.Trigger{dummyTrigger}, nil)

	// 	mockRepos := mock.NewMockRepositoryStore(controller)
	// 	mockRepos.EXPECT().Increment(gomock.Any(), dummyRepo).Return(dummyRepo, nil)

	// 	mockContents := mock.NewMockContentService(controller)
	// 	mockContents.EXPECT().Find(gomock.Any(), dummyRepo.Slug, dummyTrigger.Path, dummyHook.After).Return(dummyYaml, nil, nil)
	// 	mockContents.EXPECT().Find(gomock.Any(), dummyRepo.Slug, dummySignature.Path, dummyHook.After).Return(dummySignature, nil, nil)

	// 	mockClient := new(scm.Client)
	// 	mockClient.Contents = mockContents

	// 	mockQueue := mock.NewMockQueue(controller)
	// 	mockQueue.EXPECT().Push(gomock.Any(), gomock.Any()).Return(sql.ErrNoRows)

	// 	mockBuilds := mock.NewMockBuildStore(controller)
	// 	mockBuilds.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	// 	triggerer := New(
	// 		mockClient,
	// 		mockBuilds,
	// 		mockQueue,
	// 		mockRepos,
	// 		mockTriggers,
	// 		mockUsers,
	// 	)

	// 	builds, err := triggerer.Trigger(noContext, dummyRepo, dummyHook)
	// 	if err != sql.ErrNoRows {
	// 		t.Error("Expect error when enqueueing the build fails")
	// 	}
	// 	if got, want := len(builds), 0; got != want {
	// 		t.Errorf("Got build count %d, want %d", got, want)
	// 	}
}

var (
	dummyHook = &core.Hook{
		Event:        core.EventPush,
		Link:         "https://github.com/octocat/Hello-World/commit/7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
		Timestamp:    1299283200,
		Message:      "first commit",
		Before:       "553c2077f0edc3d5dc5d17262f6aa498e69d6f8e",
		After:        "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
		Ref:          "refs/heads/master",
		Source:       "master",
		Target:       "master",
		Author:       "octocat",
		AuthorName:   "The Octocat",
		AuthorEmail:  "octocat@hello-world.com",
		AuthorAvatar: "https://avatars3.githubusercontent.com/u/583231",
		Sender:       "octocat",
		Action:       "opened",
	}

	dummyBuild = &core.Build{
		Number: dummyRepo.Counter,
		RepoID: dummyRepo.ID,
		Status: core.StatusPending,
		Event:  core.EventPush,
		Link:   "https://github.com/octocat/Hello-World/commit/7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
		// Timestamp:    1299283200,
		Message:      "first commit",
		Before:       "553c2077f0edc3d5dc5d17262f6aa498e69d6f8e",
		After:        "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
		Ref:          "refs/heads/master",
		Source:       "master",
		Target:       "master",
		Author:       "octocat",
		AuthorName:   "The Octocat",
		AuthorEmail:  "octocat@hello-world.com",
		AuthorAvatar: "https://avatars3.githubusercontent.com/u/583231",
		Sender:       "octocat",
		Action:       "opened",
	}

	dummyRepo = &core.Repository{
		ID:         1,
		UID:        "1296269",
		UserID:     2,
		Namespace:  "octocat",
		Name:       "Hello-World",
		Slug:       "octocat/Hello-World",
		SCM:        "git",
		HTTPURL:    "https://github.com/octocat/Hello-World.git",
		SSHURL:     "git@github.com:octocat/Hello-World.git",
		Link:       "https://github.com/octocat/Hello-World",
		Branch:     "master",
		Private:    false,
		Visibility: core.VisibilityPublic,
		Active:     true,
		Counter:    42,
		Secret:     "g9dMChy22QutQM5lrpbe0yCR3f15t1gv",
		Signer:     "g9dMChy22QutQM5lrpbe0yCR3f15t1gv",
		Config:     ".drone.yml",
	}

	dummyStage = &core.Stage{
		Kind:      "pipeline",
		Type:      "docker",
		RepoID:    1,
		Name:      "default",
		Number:    1,
		OS:        "linux",
		Arch:      "amd64",
		OnSuccess: true,
		OnFailure: false,
		Status:    core.StatusPending,
	}

	dummyStages = []*core.Stage{
		dummyStage,
	}

	dummyUser = &core.User{
		ID:     2,
		Login:  "octocat",
		Active: true,
	}

	dummyYaml = &core.Config{
		Data: "kind: pipeline\nsteps: [ ]",
	}

	dummyYamlInvalid = &core.Config{
		Data: "%ERROR",
	}

	dummyYamlSkipBranch = &core.Config{
		Data: `
kind: pipeline
trigger:
  branch:
    exclude:
    - master`,
	}

	dummyYamlSkipEvent = &core.Config{
		Data: `
kind: pipeline
trigger:
  event:
    exclude:
    - push`,
	}

	dummyYamlSkipAction = &core.Config{
		Data: `
kind: pipeline
trigger:
  action:
    exclude:
    - opened`,
	}

	ignoreBuildFields = cmpopts.IgnoreFields(core.Build{},
		"Created", "Updated")

	ignoreStageFields = cmpopts.IgnoreFields(core.Stage{},
		"Created", "Updated")
)
