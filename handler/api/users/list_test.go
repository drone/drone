// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package users

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

var (
	mockUser = &core.User{
		ID:     1,
		Login:  "octocat",
		Email:  "octocat@github.com",
		Admin:  false,
		Active: true,
		Avatar: "https://avatars1.githubusercontent.com/u/583231",
	}

	mockUserList = []*core.User{
		mockUser,
	}
)

func TestHandleList(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	users := mock.NewMockUserStore(controller)
	users.EXPECT().List(gomock.Any()).Return(mockUserList, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	h := HandleList(users)

	h(w, r)
	if got, want := w.Code, 200; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	got, want := []*core.User{}, mockUserList
	json.NewDecoder(w.Body).Decode(&got)
	if diff := cmp.Diff(got, want); len(diff) > 0 {
		t.Errorf(diff)
	}
}

func TestUserList_Err(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	users := mock.NewMockUserStore(controller)
	users.EXPECT().List(gomock.Any()).Return(nil, sql.ErrNoRows)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	HandleList(users)(w, r)
	if got, want := w.Code, 500; want != got {
		t.Errorf("Want response code %d, got %d", want, got)
	}

	// got, want := new(render.Error), &render.Error{Message: "sql: no rows in result set"}
	// json.NewDecoder(w.Body).Decode(got)
	// if diff := cmp.Diff(got, want); len(diff) > 0 {
	// 	t.Errorf(diff)
	// }
}
