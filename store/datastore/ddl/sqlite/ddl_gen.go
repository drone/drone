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

package sqlite

import (
	"database/sql"
)

var migrations = []struct {
	name string
	stmt string
}{
	{
		name: "create-table-users",
		stmt: createTableUsers,
	},
	{
		name: "create-table-repos",
		stmt: createTableRepos,
	},
	{
		name: "create-table-builds",
		stmt: createTableBuilds,
	},
	{
		name: "create-index-builds-repo",
		stmt: createIndexBuildsRepo,
	},
	{
		name: "create-index-builds-author",
		stmt: createIndexBuildsAuthor,
	},
	{
		name: "create-index-builds-status",
		stmt: createIndexBuildsStatus,
	},
	{
		name: "create-table-procs",
		stmt: createTableProcs,
	},
	{
		name: "create-index-procs-build",
		stmt: createIndexProcsBuild,
	},
	{
		name: "create-table-logs",
		stmt: createTableLogs,
	},
	{
		name: "create-table-files",
		stmt: createTableFiles,
	},
	{
		name: "create-index-files-builds",
		stmt: createIndexFilesBuilds,
	},
	{
		name: "create-index-files-procs",
		stmt: createIndexFilesProcs,
	},
	{
		name: "create-table-secrets",
		stmt: createTableSecrets,
	},
	{
		name: "create-index-secrets-repo",
		stmt: createIndexSecretsRepo,
	},
	{
		name: "create-table-registry",
		stmt: createTableRegistry,
	},
	{
		name: "create-index-registry-repo",
		stmt: createIndexRegistryRepo,
	},
	{
		name: "create-table-config",
		stmt: createTableConfig,
	},
	{
		name: "create-table-tasks",
		stmt: createTableTasks,
	},
	{
		name: "create-table-agents",
		stmt: createTableAgents,
	},
	{
		name: "create-table-senders",
		stmt: createTableSenders,
	},
	{
		name: "create-index-sender-repos",
		stmt: createIndexSenderRepos,
	},
	{
		name: "alter-table-add-repo-visibility",
		stmt: alterTableAddRepoVisibility,
	},
	{
		name: "update-table-set-repo-visibility",
		stmt: updateTableSetRepoVisibility,
	},
	{
		name: "alter-table-add-repo-seq",
		stmt: alterTableAddRepoSeq,
	},
	{
		name: "update-table-set-repo-seq",
		stmt: updateTableSetRepoSeq,
	},
	{
		name: "update-table-set-repo-seq-default",
		stmt: updateTableSetRepoSeqDefault,
	},
	{
		name: "alter-table-add-repo-active",
		stmt: alterTableAddRepoActive,
	},
	{
		name: "update-table-set-repo-active",
		stmt: updateTableSetRepoActive,
	},
	{
		name: "alter-table-add-user-synced",
		stmt: alterTableAddUserSynced,
	},
	{
		name: "update-table-set-user-synced",
		stmt: updateTableSetUserSynced,
	},
	{
		name: "create-table-perms",
		stmt: createTablePerms,
	},
	{
		name: "create-index-perms-repo",
		stmt: createIndexPermsRepo,
	},
	{
		name: "create-index-perms-user",
		stmt: createIndexPermsUser,
	},
	{
		name: "alter-table-add-file-pid",
		stmt: alterTableAddFilePid,
	},
	{
		name: "alter-table-add-file-meta-passed",
		stmt: alterTableAddFileMetaPassed,
	},
	{
		name: "alter-table-add-file-meta-failed",
		stmt: alterTableAddFileMetaFailed,
	},
	{
		name: "alter-table-add-file-meta-skipped",
		stmt: alterTableAddFileMetaSkipped,
	},
	{
		name: "alter-table-update-file-meta",
		stmt: alterTableUpdateFileMeta,
	},
}

// Migrate performs the database migration. If the migration fails
// and error is returned.
func Migrate(db *sql.DB) error {
	if err := createTable(db); err != nil {
		return err
	}
	completed, err := selectCompleted(db)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	for _, migration := range migrations {
		if _, ok := completed[migration.name]; ok {

			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		{
			if _, err := tx.Exec(migration.stmt); err != nil {
				return err
			}
			if err := insertMigration(tx, migration.name); err != nil {
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func createTable(db *sql.DB) error {
	_, err := db.Exec(migrationTableCreate)
	return err
}

func insertMigration(db *sql.Tx, name string) error {
	_, err := db.Exec(migrationInsert, name)
	return err
}

func selectCompleted(db *sql.DB) (map[string]struct{}, error) {
	migrations := map[string]struct{}{}
	rows, err := db.Query(migrationSelect)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		migrations[name] = struct{}{}
	}
	return migrations, nil
}

//
// migration table ddl and sql
//

var migrationTableCreate = `
CREATE TABLE IF NOT EXISTS migrations (
 name VARCHAR(255)
,UNIQUE(name)
)
`

var migrationInsert = `
INSERT INTO migrations (name) VALUES (?)
`

var migrationSelect = `
SELECT name FROM migrations
`

//
// 001_create_table_users.sql
//

var createTableUsers = `
CREATE TABLE IF NOT EXISTS users (
 user_id     INTEGER PRIMARY KEY AUTOINCREMENT
,user_login  TEXT
,user_token  TEXT
,user_secret TEXT
,user_expiry INTEGER
,user_email  TEXT
,user_avatar TEXT
,user_active BOOLEAN
,user_admin  BOOLEAN
,user_hash   TEXT
,UNIQUE(user_login)
);
`

//
// 002_create_table_repos.sql
//

var createTableRepos = `
CREATE TABLE IF NOT EXISTS repos (
 repo_id            INTEGER PRIMARY KEY AUTOINCREMENT
,repo_user_id       INTEGER
,repo_owner         TEXT
,repo_name          TEXT
,repo_full_name     TEXT
,repo_avatar        TEXT
,repo_link          TEXT
,repo_clone         TEXT
,repo_branch        TEXT
,repo_timeout       INTEGER
,repo_private       BOOLEAN
,repo_trusted       BOOLEAN
,repo_allow_pr      BOOLEAN
,repo_allow_push    BOOLEAN
,repo_allow_deploys BOOLEAN
,repo_allow_tags    BOOLEAN
,repo_hash          TEXT
,repo_scm           TEXT
,repo_config_path   TEXT
,repo_gated         BOOLEAN
,UNIQUE(repo_full_name)
);
`

//
// 003_create_table_builds.sql
//

var createTableBuilds = `
CREATE TABLE IF NOT EXISTS builds (
 build_id        INTEGER PRIMARY KEY AUTOINCREMENT
,build_repo_id   INTEGER
,build_number    INTEGER
,build_event     TEXT
,build_status    TEXT
,build_enqueued  INTEGER
,build_created   INTEGER
,build_started   INTEGER
,build_finished  INTEGER
,build_commit    TEXT
,build_branch    TEXT
,build_ref       TEXT
,build_refspec   TEXT
,build_remote    TEXT
,build_title     TEXT
,build_message   TEXT
,build_timestamp INTEGER
,build_author    TEXT
,build_avatar    TEXT
,build_email     TEXT
,build_link      TEXT
,build_deploy    TEXT
,build_signed    BOOLEAN
,build_verified  BOOLEAN
,build_parent    INTEGER
,build_error     TEXT
,build_reviewer  TEXT
,build_reviewed  INTEGER
,build_sender    TEXT
,build_config_id INTEGER
,UNIQUE(build_number, build_repo_id)
);
`

var createIndexBuildsRepo = `
CREATE INDEX IF NOT EXISTS ix_build_repo ON builds (build_repo_id);
`

var createIndexBuildsAuthor = `
CREATE INDEX IF NOT EXISTS ix_build_author ON builds (build_author);
`

var createIndexBuildsStatus = `
CREATE INDEX IF NOT EXISTS ix_build_status_running ON builds (build_status)
WHERE build_status IN ('pending', 'running');
`

//
// 004_create_table_procs.sql
//

var createTableProcs = `
CREATE TABLE IF NOT EXISTS procs (
 proc_id         INTEGER PRIMARY KEY AUTOINCREMENT
,proc_build_id   INTEGER
,proc_pid        INTEGER
,proc_ppid       INTEGER
,proc_pgid       INTEGER
,proc_name       TEXT
,proc_state      TEXT
,proc_error      TEXT
,proc_exit_code  INTEGER
,proc_started    INTEGER
,proc_stopped    INTEGER
,proc_machine    TEXT
,proc_platform   TEXT
,proc_environ    TEXT
,UNIQUE(proc_build_id, proc_pid)
);
`

var createIndexProcsBuild = `
CREATE INDEX IF NOT EXISTS proc_build_ix ON procs (proc_build_id);
`

//
// 005_create_table_logs.sql
//

var createTableLogs = `
CREATE TABLE IF NOT EXISTS logs (
 log_id     INTEGER PRIMARY KEY AUTOINCREMENT
,log_job_id INTEGER
,log_data   BLOB
,UNIQUE(log_job_id)
);
`

//
// 006_create_table_files.sql
//

var createTableFiles = `
CREATE TABLE IF NOT EXISTS files (
 file_id       INTEGER PRIMARY KEY AUTOINCREMENT
,file_build_id INTEGER
,file_proc_id  INTEGER
,file_name     TEXT
,file_mime     TEXT
,file_size     INTEGER
,file_time     INTEGER
,file_data     BLOB
,UNIQUE(file_proc_id,file_name)
);
`

var createIndexFilesBuilds = `
CREATE INDEX IF NOT EXISTS file_build_ix ON files (file_build_id);
`

var createIndexFilesProcs = `
CREATE INDEX IF NOT EXISTS file_proc_ix  ON files (file_proc_id);
`

//
// 007_create_table_secets.sql
//

var createTableSecrets = `
CREATE TABLE IF NOT EXISTS secrets (
 secret_id          INTEGER PRIMARY KEY AUTOINCREMENT
,secret_repo_id     INTEGER
,secret_name        TEXT
,secret_value       TEXT
,secret_images      TEXT
,secret_events      TEXT
,secret_skip_verify BOOLEAN
,secret_conceal     BOOLEAN
,UNIQUE(secret_name, secret_repo_id)
);
`

var createIndexSecretsRepo = `
CREATE INDEX IF NOT EXISTS ix_secrets_repo ON secrets (secret_repo_id);
`

//
// 008_create_table_registry.sql
//

var createTableRegistry = `
CREATE TABLE IF NOT EXISTS registry (
 registry_id        INTEGER PRIMARY KEY AUTOINCREMENT
,registry_repo_id   INTEGER
,registry_addr      TEXT
,registry_username  TEXT
,registry_password  TEXT
,registry_email     TEXT
,registry_token     TEXT

,UNIQUE(registry_addr, registry_repo_id)
);
`

var createIndexRegistryRepo = `
CREATE INDEX IF NOT EXISTS ix_registry_repo ON registry (registry_repo_id);
`

//
// 009_create_table_config.sql
//

var createTableConfig = `
CREATE TABLE IF NOT EXISTS config (
 config_id       INTEGER PRIMARY KEY AUTOINCREMENT
,config_repo_id  INTEGER
,config_hash     TEXT
,config_data     BLOB
,UNIQUE(config_hash, config_repo_id)
);
`

//
// 010_create_table_tasks.sql
//

var createTableTasks = `
CREATE TABLE IF NOT EXISTS tasks (
 task_id     TEXT PRIMARY KEY
,task_data   BLOB
,task_labels BLOB
);
`

//
// 011_create_table_agents.sql
//

var createTableAgents = `
CREATE TABLE IF NOT EXISTS agents (
 agent_id       INTEGER PRIMARY KEY AUTOINCREMENT
,agent_addr     TEXT
,agent_platform TEXT
,agent_capacity INTEGER
,agent_created  INTEGER
,agent_updated  INTEGER

,UNIQUE(agent_addr)
);
`

//
// 012_create_table_senders.sql
//

var createTableSenders = `
CREATE TABLE IF NOT EXISTS senders (
 sender_id      INTEGER PRIMARY KEY AUTOINCREMENT
,sender_repo_id INTEGER
,sender_login   TEXT
,sender_allow   BOOLEAN
,sender_block   BOOLEAN

,UNIQUE(sender_repo_id,sender_login)
);
`

var createIndexSenderRepos = `
CREATE INDEX IF NOT EXISTS sender_repo_ix ON senders (sender_repo_id);
`

//
// 013_add_column_repo_visibility.sql
//

var alterTableAddRepoVisibility = `
ALTER TABLE repos ADD COLUMN repo_visibility TEXT
`

var updateTableSetRepoVisibility = `
UPDATE repos
SET repo_visibility = CASE
  WHEN repo_private = 0 THEN 'public'
  ELSE 'private'
  END
`

//
// 014_add_column_repo_seq.sql
//

var alterTableAddRepoSeq = `
ALTER TABLE repos ADD COLUMN repo_counter INTEGER;
`

var updateTableSetRepoSeq = `
UPDATE repos SET repo_counter = (
  SELECT max(build_number)
  FROM builds
  WHERE builds.build_repo_id = repos.repo_id
)
`

var updateTableSetRepoSeqDefault = `
UPDATE repos SET repo_counter = 0
WHERE repo_counter IS NULL
`

//
// 015_add_column_repo_active.sql
//

var alterTableAddRepoActive = `
ALTER TABLE repos ADD COLUMN repo_active BOOLEAN
`

var updateTableSetRepoActive = `
UPDATE repos SET repo_active = 1
`

//
// 016_add_column_user_synced.sql
//

var alterTableAddUserSynced = `
ALTER TABLE users ADD COLUMN user_synced INTEGER;
`

var updateTableSetUserSynced = `
UPDATE users SET user_synced = 0
`

//
// 017_create_table_perms.sql
//

var createTablePerms = `
CREATE TABLE IF NOT EXISTS perms (
 perm_user_id INTEGER NOT NULL
,perm_repo_id INTEGER NOT NULL
,perm_pull    BOOLEAN
,perm_push    BOOLEAN
,perm_admin   BOOLEAN
,perm_synced  INTEGER
,UNIQUE(perm_user_id, perm_repo_id)
);
`

var createIndexPermsRepo = `
CREATE INDEX IF NOT EXISTS ix_perms_repo ON perms (perm_repo_id);
`

var createIndexPermsUser = `
CREATE INDEX IF NOT EXISTS ix_perms_user ON perms (perm_user_id);
`

//
// 018_add_column_file_pid.sql
//

var alterTableAddFilePid = `
ALTER TABLE files ADD COLUMN file_pid INTEGER
`

var alterTableAddFileMetaPassed = `
ALTER TABLE files ADD COLUMN file_meta_passed INTEGER
`

var alterTableAddFileMetaFailed = `
ALTER TABLE files ADD COLUMN file_meta_failed INTEGER
`

var alterTableAddFileMetaSkipped = `
ALTER TABLE files ADD COLUMN file_meta_skipped INTEGER
`

var alterTableUpdateFileMeta = `
UPDATE files SET
 file_meta_passed=0
,file_meta_failed=0
,file_meta_skipped=0
`
