// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package build

import (
	"context"
	"database/sql"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"

	"github.com/drone/drone/store/shared/db/dbtest"
)

var noContext = context.TODO()

func TestBuild(t *testing.T) {
	conn, err := dbtest.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		dbtest.Reset(conn)
		dbtest.Disconnect(conn)
	}()

	store := New(conn).(*buildStore)
	t.Run("Create", testBuildCreate(store))
	t.Run("Purge", testBuildPurge(store))
	t.Run("Count", testBuildCount(store))
	t.Run("Pending", testBuildPending(store))
	t.Run("Running", testBuildRunning(store))
	t.Run("Latest", testBuildLatest(store))
}

func testBuildCreate(store *buildStore) func(t *testing.T) {
	return func(t *testing.T) {
		build := &core.Build{
			RepoID: 1,
			Number: 99,
			Event:  core.EventPush,
			Ref:    "refs/heads/master",
			Target: "master",
		}
		stage := &core.Stage{
			RepoID: 42,
			Number: 1,
		}
		err := store.Create(noContext, build, []*core.Stage{stage})
		if err != nil {
			t.Error(err)
		}
		if build.ID == 0 {
			t.Errorf("Want build ID assigned, got %d", build.ID)
		}
		if got, want := build.Version, int64(1); got != want {
			t.Errorf("Want build Version %d, got %d", want, got)
		}
		t.Run("Find", testBuildFind(store, build))
		t.Run("FindNumber", testBuildFindNumber(store, build))
		t.Run("FindRef", testBuildFindRef(store, build))
		t.Run("List", testBuildList(store, "", "", build, 1))
		t.Run("ListRefWithoutEvent", testBuildList(store, build.Ref, "", build, 1))
		t.Run("ListRefWithEvent", testBuildList(store, build.Ref, build.Event, build, 1))
		t.Run("ListRefWithWrongEvent", testBuildList(store, build.Ref, core.EventCron, build, 0))
		t.Run("Update", testBuildUpdate(store, build))
		t.Run("Locking", testBuildLocking(store, build))
		t.Run("Delete", testBuildDelete(store, build))
	}
}

func testBuildFind(store *buildStore, build *core.Build) func(t *testing.T) {
	return func(t *testing.T) {
		result, err := store.Find(noContext, build.ID)
		if err != nil {
			t.Error(err)
		} else {
			t.Run("Fields", testBuild(result))
		}
	}
}

func testBuildFindNumber(store *buildStore, build *core.Build) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := store.FindNumber(noContext, build.RepoID, build.Number)
		if err != nil {
			t.Error(err)
		} else {
			t.Run("Fields", testBuild(item))
		}
	}
}

func testBuildFindRef(store *buildStore, build *core.Build) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := store.FindRef(noContext, build.RepoID, build.Ref)
		if err != nil {
			t.Error(err)
		} else {
			t.Run("Fields", testBuild(item))
		}
	}
}

func testBuildList(store *buildStore, ref, event string, build *core.Build, expected int) func(t *testing.T) {
	return func(t *testing.T) {
		list, err := store.List(noContext, build.RepoID, ref, event, 10, 0)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := len(list), expected; got != want {
			t.Errorf("Want list count %d, got %d", want, got)
		} else if expected > 0 {
			t.Run("Fields", testBuild(list[0]))
		}
	}
}

func testBuildUpdate(store *buildStore, build *core.Build) func(t *testing.T) {
	return func(t *testing.T) {
		before := &core.Build{
			ID:      build.ID,
			RepoID:  build.RepoID,
			Number:  build.Number,
			Status:  core.StatusFailing,
			Version: build.Version,
		}
		err := store.Update(noContext, before)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := before.Version, build.Version+1; got != want {
			t.Errorf("Want incremented version %d, got %d", want, got)
		}
		after, err := store.Find(noContext, before.ID)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := after.Version, build.Version+1; got != want {
			t.Errorf("Want incremented version %d, got %d", want, got)
		}
		if got, want := after.Status, before.Status; got != want {
			t.Errorf("Want updated build status %v, got %v", want, got)
		}
	}
}

func testBuildLocking(store *buildStore, build *core.Build) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := store.Find(noContext, build.ID)
		if err != nil {
			t.Error(err)
			return
		}
		item.Version = 1
		err = store.Update(noContext, item)
		if err == nil {
			t.Errorf("Want Optimistic Lock Error, got nil")
		} else if err != db.ErrOptimisticLock {
			t.Errorf("Want Optimistic Lock Error")
		}
	}
}

func testBuildDelete(store *buildStore, build *core.Build) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := store.Find(noContext, build.ID)
		if err != nil {
			t.Error(err)
		}
		err = store.Delete(noContext, item)
		if err != nil {
			t.Error(err)
		}
		_, err = store.Find(noContext, item.ID)
		if want, got := sql.ErrNoRows, err; got != want {
			t.Errorf("Want %q, got %q", want, got)
		}
	}
}

func testBuildPurge(store *buildStore) func(t *testing.T) {
	return func(t *testing.T) {
		store.db.Update(func(execer db.Execer, binder db.Binder) error {
			_, err := execer.Exec("DELETE FROM builds")
			return err
		})
		store.Create(noContext, &core.Build{RepoID: 1, Number: 98}, nil)
		store.Create(noContext, &core.Build{RepoID: 1, Number: 99}, nil)
		store.Create(noContext, &core.Build{RepoID: 1, Number: 100}, nil)
		store.Create(noContext, &core.Build{RepoID: 1, Number: 101}, nil)

		before, err := store.List(noContext, 1, "", "", 100, 0)
		if err != nil {
			t.Error(err)
		}
		if got, want := len(before), 4; got != want {
			t.Errorf("Want build count %d, got %d", want, got)
		}
		err = store.Purge(noContext, 1, 100)
		if err != nil {
			t.Error(err)
		}
		after, err := store.List(noContext, 1, "", "", 100, 0)
		if err != nil {
			t.Error(err)
		}
		if got, want := len(after), 2; got != want {
			t.Errorf("Want build count %d, got %d", want, got)
		}
		for _, build := range after {
			if build.Number < 100 {
				t.Errorf("Expect purge if build number is less than 100")
			}
		}
	}
}

func testBuildCount(store *buildStore) func(t *testing.T) {
	return func(t *testing.T) {
		store.db.Update(func(execer db.Execer, binder db.Binder) error {
			_, err := execer.Exec("DELETE FROM builds")
			return err
		})
		store.Create(noContext, &core.Build{RepoID: 1, Number: 98}, nil)
		store.Create(noContext, &core.Build{RepoID: 1, Number: 99}, nil)
		store.Create(noContext, &core.Build{RepoID: 1, Number: 100}, nil)
		store.Create(noContext, &core.Build{RepoID: 1, Number: 101}, nil)

		count, err := store.Count(noContext)
		if err != nil {
			t.Error(err)
		} else if got, want := count, int64(4); got != want {
			t.Errorf("Want build count %d, got %d", want, got)
		}
	}
}

func testBuildPending(store *buildStore) func(t *testing.T) {
	return func(t *testing.T) {
		store.db.Update(func(execer db.Execer, binder db.Binder) error {
			execer.Exec("DELETE FROM builds")
			execer.Exec("DELETE FROM stages")
			return nil
		})
		store.Create(noContext, &core.Build{RepoID: 1, Number: 98, Status: core.StatusPending}, []*core.Stage{&core.Stage{RepoID: 1, Number: 1, Status: core.StatusPending}})
		store.Create(noContext, &core.Build{RepoID: 1, Number: 99, Status: core.StatusPending}, []*core.Stage{&core.Stage{RepoID: 1, Number: 1, Status: core.StatusPending}})
		store.Create(noContext, &core.Build{RepoID: 1, Number: 100, Status: core.StatusRunning}, []*core.Stage{&core.Stage{RepoID: 1, Number: 1, Status: core.StatusRunning}})
		store.Create(noContext, &core.Build{RepoID: 1, Number: 101, Status: core.StatusPassing}, []*core.Stage{&core.Stage{RepoID: 1, Number: 1, Status: core.StatusPassing}})

		count, err := store.Count(noContext)
		if err != nil {
			t.Error(err)
		} else if got, want := count, int64(4); got != want {
			t.Errorf("Want build count %d, got %d", want, got)
		}
		list, err := store.Pending(noContext)
		if err != nil {
			t.Error(err)
		} else if got, want := len(list), 2; got != want {
			t.Errorf("Want list count %d, got %d", want, got)
		}
	}
}

func testBuildRunning(store *buildStore) func(t *testing.T) {
	return func(t *testing.T) {
		store.db.Update(func(execer db.Execer, binder db.Binder) error {
			execer.Exec("DELETE FROM builds")
			execer.Exec("DELETE FROM stages")
			return nil
		})
		store.Create(noContext, &core.Build{RepoID: 1, Number: 98, Status: core.StatusRunning}, []*core.Stage{&core.Stage{RepoID: 1, Number: 1, Status: core.StatusRunning}})
		store.Create(noContext, &core.Build{RepoID: 1, Number: 99, Status: core.StatusRunning}, []*core.Stage{&core.Stage{RepoID: 1, Number: 1, Status: core.StatusRunning}})
		store.Create(noContext, &core.Build{RepoID: 1, Number: 100, Status: core.StatusBlocked}, []*core.Stage{&core.Stage{RepoID: 1, Number: 1, Status: core.StatusBlocked}})
		store.Create(noContext, &core.Build{RepoID: 1, Number: 101, Status: core.StatusPassing}, []*core.Stage{&core.Stage{RepoID: 1, Number: 1, Status: core.StatusPassing}})

		count, err := store.Count(noContext)
		if err != nil {
			t.Error(err)
		} else if got, want := count, int64(4); got != want {
			t.Errorf("Want build count %d, got %d", want, got)
		}
		list, err := store.Running(noContext)
		if err != nil {
			t.Error(err)
		} else if got, want := len(list), 2; got != want {
			t.Errorf("Want list count %d, got %d", want, got)
		}
	}
}

func testBuildLatest(store *buildStore) func(t *testing.T) {
	return func(t *testing.T) {
		store.db.Update(func(execer db.Execer, binder db.Binder) error {
			execer.Exec("DELETE FROM stages")
			execer.Exec("DELETE FROM latest")
			execer.Exec("DELETE FROM builds")
			return nil
		})

		//
		// step 1: insert the initial builds
		//

		build := &core.Build{
			RepoID: 1,
			Number: 99,
			Event:  core.EventPush,
			Ref:    "refs/heads/master",
			Target: "master",
		}

		err := store.Create(noContext, build, []*core.Stage{})
		if err != nil {
			t.Error(err)
			return
		}

		develop := &core.Build{
			RepoID: 1,
			Number: 100,
			Event:  core.EventPush,
			Ref:    "refs/heads/develop",
			Target: "develop",
		}
		err = store.Create(noContext, develop, []*core.Stage{})
		if err != nil {
			t.Error(err)
			return
		}

		err = store.Create(noContext, &core.Build{
			RepoID: 1,
			Number: 999,
			Event:  core.EventPullRequest,
			Ref:    "refs/pulls/10/head",
			Source: "develop",
			Target: "master",
		}, []*core.Stage{})
		if err != nil {
			t.Error(err)
			return
		}

		//
		// step 2: verify the latest build number was captured
		//

		latest, _ := store.LatestBranches(noContext, build.RepoID)
		if len(latest) != 2 {
			t.Errorf("Expect latest branch list == 1, got %d", len(latest))
			return
		}
		if got, want := latest[0].Number, build.Number; got != want {
			t.Errorf("Expected latest master build number %d, got %d", want, got)
		}
		if got, want := latest[1].Number, develop.Number; got != want {
			t.Errorf("Expected latest develop build number %d, got %d", want, got)
			return
		}

		build = &core.Build{
			RepoID: 1,
			Number: 101,
			Event:  core.EventPush,
			Ref:    "refs/heads/master",
			Target: "master",
		}
		err = store.Create(noContext, build, []*core.Stage{})
		if err != nil {
			t.Error(err)
			return
		}

		latest, _ = store.LatestBranches(noContext, build.RepoID)
		if len(latest) != 2 {
			t.Errorf("Expect latest branch list == 1")
			return
		}
		if got, want := latest[1].Number, build.Number; got != want {
			t.Errorf("Expected latest build number %d, got %d", want, got)
			return
		}

		err = store.DeleteBranch(noContext, build.RepoID, build.Target)
		if err != nil {
			t.Error(err)
			return
		}

		latest, _ = store.LatestBranches(noContext, build.RepoID)
		if len(latest) != 1 {
			t.Errorf("Expect latest branch list == 1 after delete")
			return
		}
	}
}

func testBuild(item *core.Build) func(t *testing.T) {
	return func(t *testing.T) {
		if got, want := item.RepoID, int64(1); got != want {
			t.Errorf("Want build repo ID %d, got %d", want, got)
		}
		if got, want := item.Number, int64(99); got != want {
			t.Errorf("Want build number %d, got %d", want, got)
		}
		if got, want := item.Ref, "refs/heads/master"; got != want {
			t.Errorf("Want build ref %q, got %q", want, got)
		}
	}
}
