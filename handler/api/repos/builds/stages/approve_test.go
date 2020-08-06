// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package stages

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/errors"
	"github.com/drone/drone/mock"

	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

func TestApprove(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusPending,
	}
	mockStage := &core.Stage{
		ID:     222,
		Number: 2,
		Status: core.StatusBlocked,
		OS:     "linux",
		Arch:   "arm",
	}

	checkStage := func(_ context.Context, stage *core.Stage) error {
		if stage.Status != core.StatusPending {
			t.Errorf("Want stage status changed to Pending")
		}
		return nil
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().FindNumber(gomock.Any(), mockBuild.ID, mockStage.Number).Return(mockStage, nil)
	stages.EXPECT().Update(gomock.Any(), mockStage).Return(nil).Do(checkStage)

	sched := mock.NewMockScheduler(controller)
	sched.EXPECT().Schedule(gomock.Any(), mockStage).Return(nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "2")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleApprove(repos, builds, stages, sched)(w, r)
	if got, want := w.Code, 204; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}
}

// this test verifies that a 400 bad request status is returned
// from the http.Handler with a human-readable error message if
// the build status is not Blocked.
func TestApprove_InvalidStatus(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusPending,
	}
	mockStage := &core.Stage{
		ID:     222,
		Number: 2,
		Status: core.StatusPending,
		OS:     "linux",
		Arch:   "arm",
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().FindNumber(gomock.Any(), mockBuild.ID, mockStage.Number).Return(mockStage, nil)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "2")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleApprove(repos, builds, stages, nil)(w, r)
	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.New(`Cannot approve a Pipeline with Status "pending"`)
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// this test verifies that a 400 bad request status is returned
// from the http.Handler with a human-readable error message if
// the build number url parameter fails to parse.
func TestApprove_InvalidBuildNumber(t *testing.T) {
	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "I")
	c.URLParams.Add("stage", "2")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleApprove(nil, nil, nil, nil)(w, r)
	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.New("Invalid build number")
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// this test verifies that a 400 bad request status is returned
// from the http.Handler with a human-readable error message if
// the stage number url parameter fails to parse.
func TestApprove_InvalidStageNumber(t *testing.T) {
	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "II")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleApprove(nil, nil, nil, nil)(w, r)
	if got, want := w.Code, 400; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.New("Invalid stage number")
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// this test verifies that a 404 not found status is returned
// from the http.Handler with a human-readable error message if
// the stage is not found in the database.
func TestApprove_StageNotFound(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusPending,
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().FindNumber(gomock.Any(), mockBuild.ID, 2).Return(nil, sql.ErrNoRows)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "2")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleApprove(repos, builds, stages, nil)(w, r)
	if got, want := w.Code, 404; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.New("Stage not found")
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// this test verifies that a 404 not found status is returned
// from the http.Handler with a human-readable error message if
// the build is not found in the database.
func TestApprove_BuildNotFound(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		Namespace: "octocat",
		Name:      "hello-world",
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, int64(1)).Return(nil, sql.ErrNoRows)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "2")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleApprove(repos, builds, nil, nil)(w, r)
	if got, want := w.Code, 404; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.New("Build not found")
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// this test verifies that a 404 not found status is returned
// from the http.Handler with a human-readable error message if
// the repository is not found in the database.
func TestApprove_RepoNotFound(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		Namespace: "octocat",
		Name:      "hello-world",
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(nil, sql.ErrNoRows)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "2")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleApprove(repos, nil, nil, nil)(w, r)
	if got, want := w.Code, 404; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.New("Repository not found")
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// this test verifies that a 500 internal error status is returned
// from the http.Handler with a human-readable error message if
// the the server fails to persist stage updates to the database.
func TestApprove_CannotSaveStage(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusPending,
	}
	mockStage := &core.Stage{
		ID:     222,
		Number: 2,
		Status: core.StatusBlocked,
		OS:     "linux",
		Arch:   "arm",
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().FindNumber(gomock.Any(), mockBuild.ID, mockStage.Number).Return(mockStage, nil)
	stages.EXPECT().Update(gomock.Any(), mockStage).Return(sql.ErrConnDone)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "2")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleApprove(repos, builds, stages, nil)(w, r)
	if got, want := w.Code, 500; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.New("There was a problem approving the Pipeline")
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}

// this test verifies that a 500 internal error status is returned
// from the http.Handler with a human-readable error message if
// the the server fails to enqueue the approved pipeline.
func TestApprove_CannotEnqueue(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	mockRepo := &core.Repository{
		Namespace: "octocat",
		Name:      "hello-world",
	}
	mockBuild := &core.Build{
		ID:     111,
		Number: 1,
		Status: core.StatusPending,
	}
	mockStage := &core.Stage{
		ID:     222,
		Number: 2,
		Status: core.StatusBlocked,
		OS:     "linux",
		Arch:   "arm",
	}

	repos := mock.NewMockRepositoryStore(controller)
	repos.EXPECT().FindName(gomock.Any(), mockRepo.Namespace, mockRepo.Name).Return(mockRepo, nil)

	builds := mock.NewMockBuildStore(controller)
	builds.EXPECT().FindNumber(gomock.Any(), mockRepo.ID, mockBuild.Number).Return(mockBuild, nil)

	stages := mock.NewMockStageStore(controller)
	stages.EXPECT().FindNumber(gomock.Any(), mockBuild.ID, mockStage.Number).Return(mockStage, nil)
	stages.EXPECT().Update(gomock.Any(), mockStage).Return(nil)

	sched := mock.NewMockScheduler(controller)
	sched.EXPECT().Schedule(gomock.Any(), gomock.Any()).Return(io.EOF)

	c := new(chi.Context)
	c.URLParams.Add("owner", "octocat")
	c.URLParams.Add("name", "hello-world")
	c.URLParams.Add("number", "1")
	c.URLParams.Add("stage", "2")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(
		context.WithValue(context.Background(), chi.RouteCtxKey, c),
	)

	HandleApprove(repos, builds, stages, sched)(w, r)
	if got, want := w.Code, 500; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := new(errors.Error), errors.New("There was a problem scheduling the Pipeline")
	json.NewDecoder(w.Body).Decode(got)
	if diff := cmp.Diff(got, want); len(diff) != 0 {
		t.Errorf(diff)
	}
}
