// Copyright 2018 Drone.IO Inc.
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

package datastore

import (
	"fmt"
	"time"

	"github.com/drone/drone/model"
	"github.com/drone/drone/store/datastore/sql"
	"github.com/russross/meddler"
)

func (db *datastore) GetBuild(id int64) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.Load(db, buildTable, build, id)
	return build, err
}

func (db *datastore) GetBuildNumber(repo *model.Repo, num int) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildNumberQuery), repo.ID, num)
	return build, err
}

func (db *datastore) GetBuildRef(repo *model.Repo, ref string) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildRefQuery), repo.ID, ref)
	return build, err
}

func (db *datastore) GetBuildCommit(repo *model.Repo, sha, branch string) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildCommitQuery), repo.ID, sha, branch)
	return build, err
}

func (db *datastore) GetBuildLast(repo *model.Repo, branch string) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildLastQuery), repo.ID, branch)
	return build, err
}

func (db *datastore) GetBuildLastBefore(repo *model.Repo, branch string, num int64) (*model.Build, error) {
	var build = new(model.Build)
	var err = meddler.QueryRow(db, build, rebind(buildLastBeforeQuery), repo.ID, branch, num)
	return build, err
}

func (db *datastore) GetBuildList(repo *model.Repo, page int) ([]*model.Build, error) {
	var builds = []*model.Build{}
	var err = meddler.QueryAll(db, &builds, rebind(buildListQuery), repo.ID, 50*(page-1))
	return builds, err
}

func (db *datastore) GetBuildQueue() ([]*model.Feed, error) {
	feed := []*model.Feed{}
	err := meddler.QueryAll(db, &feed, buildQueueList)
	return feed, err
}

func (db *datastore) GetRecentBuilds() ([]*model.Feed, error) {
	feed := []*model.Feed{}
	err := meddler.QueryAll(db, &feed, buildRecentList)
	return feed, err
}

func (db *datastore) CreateBuild(build *model.Build, procs ...*model.Proc) error {
	id, err := db.incrementRepoRetry(build.RepoID)
	if err != nil {
		return err
	}
	build.Number = id
	build.Created = time.Now().UTC().Unix()
	build.Enqueued = build.Created
	err = meddler.Insert(db, buildTable, build)
	if err != nil {
		return err
	}
	for _, proc := range procs {
		proc.BuildID = build.ID
		err = meddler.Insert(db, "procs", proc)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *datastore) incrementRepoRetry(id int64) (int, error) {
	repo, err := db.GetRepo(id)
	if err != nil {
		return 0, fmt.Errorf("database: cannot fetch repository: %s", err)
	}
	for i := 0; i < 10; i++ {
		seq, err := db.incrementRepo(repo.ID, repo.Counter+i, repo.Counter+i+1)
		if err != nil {
			return 0, err
		}
		if seq == 0 {
			continue
		}
		return seq, nil
	}
	return 0, fmt.Errorf("cannot increment next build number")
}

func (db *datastore) incrementRepo(id int64, old, new int) (int, error) {
	results, err := db.Exec(sql.Lookup(db.driver, "repo-update-counter"), new, old, id)
	if err != nil {
		return 0, fmt.Errorf("database: update repository counter: %s", err)
	}
	updated, err := results.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("database: update repository counter: %s", err)
	}
	if updated == 0 {
		return 0, nil
	}
	return new, nil
}

func (db *datastore) UpdateBuild(build *model.Build) error {
	return meddler.Update(db, buildTable, build)
}

func (db *datastore) GetBuildCount() (count int, err error) {
	err = db.QueryRow(
		sql.Lookup(db.driver, "count-builds"),
	).Scan(&count)
	return
}

const buildTable = "builds"

const buildListQuery = `
SELECT *
FROM builds
WHERE build_repo_id = ?
ORDER BY build_number DESC
LIMIT 50 OFFSET ?
`

const buildNumberQuery = `
SELECT *
FROM builds
WHERE build_repo_id = ?
  AND build_number = ?
LIMIT 1;
`

const buildLastQuery = `
SELECT *
FROM builds
WHERE build_repo_id = ?
  AND build_branch  = ?
  AND build_event   = 'push'
ORDER BY build_number DESC
LIMIT 1
`

const buildLastBeforeQuery = `
SELECT *
FROM builds
WHERE build_repo_id = ?
  AND build_branch  = ?
  AND build_id < ?
ORDER BY build_number DESC
LIMIT 1
`

const buildCommitQuery = `
SELECT *
FROM builds
WHERE build_repo_id = ?
  AND build_commit  = ?
  AND build_branch  = ?
LIMIT 1
`

const buildRefQuery = `
SELECT *
FROM builds
WHERE build_repo_id = ?
  AND build_ref     = ?
LIMIT 1
`

const buildNumberLast = `
SELECT MAX(build_number)
FROM builds
WHERE build_repo_id = ?
`

const buildQueueList = `
SELECT
 repo_owner
,repo_name
,repo_full_name
,build_number
,build_event
,build_status
,build_created
,build_started
,build_finished
,build_commit
,build_branch
,build_ref
,build_refspec
,build_remote
,build_title
,build_message
,build_author
,build_email
,build_avatar
FROM
 builds b
,repos r
WHERE b.build_repo_id = r.repo_id
  AND b.build_status IN ('pending','running')
`

const buildRecentList = `
SELECT
 repo_owner
,repo_name
,repo_full_name
,build_number
,build_event
,build_status
,build_created
,build_started
,build_finished
,build_commit
,build_branch
,build_ref
,build_refspec
,build_remote
,build_title
,build_message
,build_author
,build_email
,build_avatar
FROM
 builds b
,repos r
WHERE b.build_repo_id = r.repo_id
ORDER BY b.build_number DESC
LIMIT 50
`
