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

package build

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
)

// regular expression to extract the pull request number
// from the git ref (e.g. refs/pulls/{d}/head)
var pr = regexp.MustCompile("\\d+")

// New returns a new Buildcore.
func New(db *db.DB) core.BuildStore {
	return &buildStore{db}
}

type buildStore struct {
	db *db.DB
}

// Find returns a build from the datacore.
func (s *buildStore) Find(ctx context.Context, id int64) (*core.Build, error) {
	out := &core.Build{ID: id}
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := toParams(out)
		query, args, err := binder.BindNamed(queryKey, params)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRow(row, out)
	})
	return out, err
}

// FindNumber returns a build from the datastore by build number.
func (s *buildStore) FindNumber(ctx context.Context, repo, number int64) (*core.Build, error) {
	out := &core.Build{Number: number, RepoID: repo}
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := toParams(out)
		query, args, err := binder.BindNamed(queryNumber, params)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRow(row, out)
	})
	return out, err
}

// FindLast returns the last build from the datastore by ref.
func (s *buildStore) FindRef(ctx context.Context, repo int64, ref string) (*core.Build, error) {
	out := &core.Build{RepoID: repo, Ref: ref}
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := toParams(out)
		query, args, err := binder.BindNamed(queryRowRef, params)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRow(row, out)
	})
	return out, err
}

// List returns a list of builds from the datastore by repository id.
func (s *buildStore) List(ctx context.Context, repo int64, ref, event string, limit, offset int) ([]*core.Build, error) {
	var out []*core.Build
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{
			"build_repo_id": repo,
			"build_ref":     ref,
			"build_event":   event,
			"limit":         limit,
			"offset":        offset,
		}
		stmt, args, err := binder.BindNamed(queryRepo, params)
		if err != nil {
			return err
		}
		rows, err := queryer.Query(stmt, args...)
		if err != nil {
			return err
		}
		out, err = scanRows(rows)
		return err
	})
	return out, err
}

// LatestBranches returns a list of the latest build by branch.
func (s *buildStore) LatestBranches(ctx context.Context, repo int64) ([]*core.Build, error) {
	return s.latest(ctx, repo, "branch")
}

// LatestPulls returns a list of the latest builds by pull requests.
func (s *buildStore) LatestPulls(ctx context.Context, repo int64) ([]*core.Build, error) {
	return s.latest(ctx, repo, "pull_request")
}

// LatestDeploys returns a list of the latest builds by target deploy.
func (s *buildStore) LatestDeploys(ctx context.Context, repo int64) ([]*core.Build, error) {
	return s.latest(ctx, repo, "deployment")
}

func (s *buildStore) latest(ctx context.Context, repo int64, event string) ([]*core.Build, error) {
	var out []*core.Build
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{
			"latest_repo_id": repo,
			"latest_type":    event,
		}
		stmt, args, err := binder.BindNamed(queryLatestList, params)
		if err != nil {
			return err
		}
		rows, err := queryer.Query(stmt, args...)
		if err != nil {
			return err
		}
		out, err = scanRows(rows)
		return err
	})
	return out, err
}

// Pending returns a list of pending builds from the datastore by repository id.
func (s *buildStore) Pending(ctx context.Context) ([]*core.Build, error) {
	var out []*core.Build
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		rows, err := queryer.Query(queryPending)
		if err != nil {
			return err
		}
		out, err = scanRows(rows)
		return err
	})
	return out, err
}

// Running returns a list of running builds from the datastore by repository id.
func (s *buildStore) Running(ctx context.Context) ([]*core.Build, error) {
	var out []*core.Build
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		rows, err := queryer.Query(queryRunning)
		if err != nil {
			return err
		}
		out, err = scanRows(rows)
		return err
	})
	return out, err
}

// Create persists a build to the datacore.
func (s *buildStore) Create(ctx context.Context, build *core.Build, stages []*core.Stage) error {
	var err error
	switch s.db.Driver() {
	case db.Postgres:
		err = s.createPostgres(ctx, build, stages)
	default:
		err = s.create(ctx, build, stages)
	}
	if err != nil {
		return err
	}
	var event, name string
	switch build.Event {
	case core.EventPullRequest:
		event = "pull_request"
		name = pr.FindString(build.Ref)
	case core.EventPush:
		event = "branch"
		name = build.Target
	case core.EventPromote, core.EventRollback:
		event = "deployment"
		name = build.Deploy
	default:
		return nil
	}
	return s.index(ctx, build.ID, build.RepoID, event, name)
}

func (s *buildStore) create(ctx context.Context, build *core.Build, stages []*core.Stage) error {
	build.Version = 1
	return s.db.Update(func(execer db.Execer, binder db.Binder) error {
		params := toParams(build)
		stmt, args, err := binder.BindNamed(stmtInsert, params)
		if err != nil {
			return err
		}
		res, err := execer.Exec(stmt, args...)
		if err != nil {
			return err
		}
		build.ID, err = res.LastInsertId()
		if err != nil {
			return err
		}

		for _, stage := range stages {
			stage.Version = 1
			stage.BuildID = build.ID
			params := toStageParams(stage)
			stmt, args, err := binder.BindNamed(stmtStageInsert, params)
			if err != nil {
				return err
			}
			res, err := execer.Exec(stmt, args...)
			if err != nil {
				return err
			}
			stage.ID, err = res.LastInsertId()
		}
		return err
	})
}

func (s *buildStore) createPostgres(ctx context.Context, build *core.Build, stages []*core.Stage) error {
	build.Version = 1
	return s.db.Update(func(execer db.Execer, binder db.Binder) error {
		params := toParams(build)
		stmt, args, err := binder.BindNamed(stmtInsertPg, params)
		if err != nil {
			return err
		}
		err = execer.QueryRow(stmt, args...).Scan(&build.ID)
		if err != nil {
			return err
		}

		for _, stage := range stages {
			stage.Version = 1
			stage.BuildID = build.ID
			params := toStageParams(stage)
			stmt, args, err := binder.BindNamed(stmtStageInsertPg, params)
			if err != nil {
				return err
			}
			err = execer.QueryRow(stmt, args...).Scan(&stage.ID)
			if err != nil {
				return err
			}
		}
		return err
	})
}

// Update updates a build in the datacore.
func (s *buildStore) Update(ctx context.Context, build *core.Build) error {
	versionNew := build.Version + 1
	versionOld := build.Version

	err := s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := toParams(build)
		params["build_version_old"] = versionOld
		params["build_version_new"] = versionNew
		stmt, args, err := binder.BindNamed(stmtUpdate, params)
		if err != nil {
			return err
		}
		res, err := execer.Exec(stmt, args...)
		if err != nil {
			return err
		}
		effected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if effected == 0 {
			return db.ErrOptimisticLock
		}
		return nil
	})
	if err == nil {
		build.Version = versionNew
	}
	return err
}

func (s *buildStore) index(ctx context.Context, build, repo int64, event, name string) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := map[string]interface{}{
			"latest_repo_id":  repo,
			"latest_build_id": build,
			"latest_type":     event,
			"latest_name":     name,
			"latest_created":  time.Now().Unix(),
			"latest_updated":  time.Now().Unix(),
			"latest_deleted":  time.Now().Unix(),
		}
		stmtInsert := stmtInsertLatest
		switch s.db.Driver() {
		case db.Postgres:
			stmtInsert = stmtInsertLatestPg
		case db.Mysql:
			stmtInsert = stmtInsertLatestMysql
		}
		stmt, args, err := binder.BindNamed(stmtInsert, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

// Delete deletes a build from the datacore.
func (s *buildStore) Delete(ctx context.Context, build *core.Build) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := toParams(build)
		stmt, args, err := binder.BindNamed(stmtDelete, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

// DeletePull deletes a pull request index from the datastore.
func (s *buildStore) DeletePull(ctx context.Context, repo int64, number int) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := map[string]interface{}{
			"latest_repo_id": repo,
			"latest_name":    fmt.Sprint(number),
			"latest_type":    "pull_request",
		}
		stmt, args, err := binder.BindNamed(stmtDeleteLatest, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

// DeleteBranch deletes a branch index from the datastore.
func (s *buildStore) DeleteBranch(ctx context.Context, repo int64, branch string) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := map[string]interface{}{
			"latest_repo_id": repo,
			"latest_name":    branch,
			"latest_type":    "branch",
		}
		stmt, args, err := binder.BindNamed(stmtDeleteLatest, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

// DeleteDeploy deletes a deploy index from the datastore.
func (s *buildStore) DeleteDeploy(ctx context.Context, repo int64, environment string) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := map[string]interface{}{
			"latest_repo_id": repo,
			"latest_name":    environment,
			"latest_type":    "deployment",
		}
		stmt, args, err := binder.BindNamed(stmtDeleteLatest, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

// Purge deletes builds from the database where the build number is less than n.
func (s *buildStore) Purge(ctx context.Context, repo, number int64) error {
	build := &core.Build{
		RepoID: repo,
		Number: number,
	}
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := toParams(build)
		stmt, args, err := binder.BindNamed(stmtPurge, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

// Count returns a count of builds.
func (s *buildStore) Count(ctx context.Context) (i int64, err error) {
	err = s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		return queryer.QueryRow(queryCount).Scan(&i)
	})
	return
}

const queryCount = `
SELECT COUNT(*)
FROM builds
`

const queryBase = `
SELECT
 build_id
,build_repo_id
,build_trigger
,build_number
,build_parent
,build_status
,build_error
,build_event
,build_action
,build_link
,build_timestamp
,build_title
,build_message
,build_before
,build_after
,build_ref
,build_source_repo
,build_source
,build_target
,build_author
,build_author_name
,build_author_email
,build_author_avatar
,build_sender
,build_params
,build_cron
,build_deploy
,build_deploy_id
,build_debug
,build_started
,build_finished
,build_created
,build_updated
,build_version
`

const queryKey = queryBase + `
FROM builds
WHERE build_id = :build_id
`

const queryNumber = queryBase + `
FROM builds
WHERE build_repo_id = :build_repo_id
  AND build_number = :build_number
`

const queryRowRef = queryBase + `
FROM builds
WHERE build_repo_id = :build_repo_id
  AND build_ref = :build_ref
ORDER BY build_id DESC
LIMIT 1
`

const queryRepo = queryBase + `
FROM builds
WHERE build_repo_id = :build_repo_id
	AND ('' = :build_ref   OR build_ref = :build_ref)
	AND ('' = :build_event OR build_event = :build_event)
ORDER BY build_id DESC
LIMIT :limit OFFSET :offset
`

const queryPending = queryBase + `
FROM builds
WHERE EXISTS (
    SELECT stage_id
    FROM stages
    WHERE stages.stage_build_id = builds.build_id
    AND stages.stage_status = 'pending'
)
ORDER BY build_id ASC
`

const queryRunning = queryBase + `
FROM builds
WHERE EXISTS (
    SELECT stage_id
    FROM stages
    WHERE stages.stage_build_id = builds.build_id
    AND stages.stage_status = 'running'
)
ORDER BY build_id ASC
`

// const queryRunningOLD = queryBase + `
// FROM builds
// WHERE build_status = 'running'
// ORDER BY build_id ASC
// `

const queryAll = queryBase + `
FROM builds
WHERE build_id > :build_id
LIMIT :limit OFFSET :offset
`

const stmtUpdate = `
UPDATE builds SET
 build_parent = :build_parent
,build_status = :build_status
,build_error = :build_error
,build_event = :build_event
,build_action = :build_action
,build_link = :build_link
,build_timestamp = :build_timestamp 
,build_title = :build_title
,build_message = :build_message
,build_before = :build_before
,build_after = :build_after
,build_ref = :build_ref
,build_source_repo = :build_source_repo
,build_source = :build_source
,build_target = :build_target
,build_author = :build_author
,build_author_name = :build_author_name
,build_author_email = :build_author_email
,build_author_avatar = :build_author_avatar
,build_sender = :build_sender
,build_params = :build_params
,build_cron = :build_cron
,build_deploy = :build_deploy
,build_started = :build_started
,build_finished = :build_finished
,build_updated = :build_updated
,build_version = :build_version_new
WHERE build_id = :build_id
  AND build_version = :build_version_old
`

const stmtInsert = `
INSERT INTO builds (
 build_repo_id
,build_trigger
,build_number
,build_parent
,build_status
,build_error
,build_event
,build_action
,build_link
,build_timestamp
,build_title
,build_message
,build_before
,build_after
,build_ref
,build_source_repo
,build_source
,build_target
,build_author
,build_author_name
,build_author_email
,build_author_avatar
,build_sender
,build_params
,build_cron
,build_deploy
,build_deploy_id
,build_debug
,build_started
,build_finished
,build_created
,build_updated
,build_version
) VALUES (
 :build_repo_id
,:build_trigger
,:build_number
,:build_parent
,:build_status
,:build_error
,:build_event
,:build_action
,:build_link
,:build_timestamp
,:build_title
,:build_message
,:build_before
,:build_after
,:build_ref
,:build_source_repo
,:build_source
,:build_target
,:build_author
,:build_author_name
,:build_author_email
,:build_author_avatar
,:build_sender
,:build_params
,:build_cron
,:build_deploy
,:build_deploy_id
,:build_debug
,:build_started
,:build_finished
,:build_created
,:build_updated
,:build_version
)
`

const stmtInsertPg = stmtInsert + `
RETURNING build_id
`

const stmtStageInsert = `
INSERT INTO stages (
 stage_repo_id
,stage_build_id
,stage_number
,stage_name
,stage_kind
,stage_type
,stage_status
,stage_error
,stage_errignore
,stage_exit_code
,stage_limit
,stage_limit_repo
,stage_os
,stage_arch
,stage_variant
,stage_kernel
,stage_machine
,stage_started
,stage_stopped
,stage_created
,stage_updated
,stage_version
,stage_on_success
,stage_on_failure
,stage_depends_on
,stage_labels
) VALUES (
 :stage_repo_id
,:stage_build_id
,:stage_number
,:stage_name
,:stage_kind
,:stage_type
,:stage_status
,:stage_error
,:stage_errignore
,:stage_exit_code
,:stage_limit
,:stage_limit_repo
,:stage_os
,:stage_arch
,:stage_variant
,:stage_kernel
,:stage_machine
,:stage_started
,:stage_stopped
,:stage_created
,:stage_updated
,:stage_version
,:stage_on_success
,:stage_on_failure
,:stage_depends_on
,:stage_labels
)
`

const stmtStageInsertPg = stmtStageInsert + `
RETURNING stage_id
`

const stmtDelete = `
DELETE FROM builds
WHERE build_id = :build_id
`

const stmtPurge = `
DELETE FROM builds
WHERE build_repo_id = :build_repo_id
AND build_number < :build_number
`

//
// latest builds index
//

const stmtInsertLatest = `
INSERT INTO latest (
 latest_repo_id
,latest_build_id
,latest_type
,latest_name
,latest_created
,latest_updated
,latest_deleted
) VALUES (
 :latest_repo_id
,:latest_build_id
,:latest_type
,:latest_name
,:latest_created
,:latest_updated
,:latest_deleted
) ON CONFLICT (latest_repo_id, latest_type, latest_name)
DO UPDATE SET latest_build_id = EXCLUDED.latest_build_id
`

const stmtInsertLatestPg = `
INSERT INTO latest (
 latest_repo_id
,latest_build_id
,latest_type
,latest_name
,latest_created
,latest_updated
,latest_deleted
) VALUES (
 :latest_repo_id
,:latest_build_id
,:latest_type
,:latest_name
,:latest_created
,:latest_updated
,:latest_deleted
) ON CONFLICT (latest_repo_id, latest_type, latest_name)
DO UPDATE SET latest_build_id = EXCLUDED.latest_build_id
`

const stmtInsertLatestMysql = `
INSERT INTO latest (
 latest_repo_id
,latest_build_id
,latest_type
,latest_name
,latest_created
,latest_updated
,latest_deleted
) VALUES (
 :latest_repo_id
,:latest_build_id
,:latest_type
,:latest_name
,:latest_created
,:latest_updated
,:latest_deleted
) ON DUPLICATE KEY UPDATE latest_build_id = :latest_build_id
`

const stmtDeleteLatest = `
DELETE FROM latest
WHERE latest_repo_id  = :latest_repo_id
  AND latest_type     = :latest_type
  AND latest_name     = :latest_name
`

const queryLatestList = queryBase + `
FROM builds
WHERE build_id IN (
	SELECT latest_build_id
	FROM latest
	WHERE latest_repo_id  = :latest_repo_id
	  AND latest_type     = :latest_type
)
`
