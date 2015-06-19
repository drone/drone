package builtin

// DO NOT EDIT
// code generated by go:generate

import (
	"database/sql"
	"encoding/json"

	. "github.com/drone/drone/pkg/types"
)

var _ = json.Marshal

// generic database interface, matching both *sql.Db and *sql.Tx
type jobDB interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

func getJob(db jobDB, query string, args ...interface{}) (*Job, error) {
	row := db.QueryRow(query, args...)
	return scanJob(row)
}

func getJobs(db jobDB, query string, args ...interface{}) ([]*Job, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func createJob(db jobDB, query string, v *Job) error {
	var v0 int64
	var v1 int
	var v2 string
	var v3 int
	var v4 int64
	var v5 int64
	var v6 []byte
	v0 = v.BuildID
	v1 = v.Number
	v2 = v.Status
	v3 = v.ExitCode
	v4 = v.Started
	v5 = v.Finished
	v6, _ = json.Marshal(v.Environment)

	res, err := db.Exec(query,
		&v0,
		&v1,
		&v2,
		&v3,
		&v4,
		&v5,
		&v6,
	)
	if err != nil {
		return err
	}

	v.ID, err = res.LastInsertId()
	return err
}

func updateJob(db jobDB, query string, v *Job) error {
	var v0 int64
	var v1 int64
	var v2 int
	var v3 string
	var v4 int
	var v5 int64
	var v6 int64
	var v7 []byte
	v0 = v.ID
	v1 = v.BuildID
	v2 = v.Number
	v3 = v.Status
	v4 = v.ExitCode
	v5 = v.Started
	v6 = v.Finished
	v7, _ = json.Marshal(v.Environment)

	_, err := db.Exec(query,
		&v1,
		&v2,
		&v3,
		&v4,
		&v5,
		&v6,
		&v7,
		&v0,
	)
	return err
}

func scanJob(row *sql.Row) (*Job, error) {
	var v0 int64
	var v1 int64
	var v2 int
	var v3 string
	var v4 int
	var v5 int64
	var v6 int64
	var v7 []byte

	err := row.Scan(
		&v0,
		&v1,
		&v2,
		&v3,
		&v4,
		&v5,
		&v6,
		&v7,
	)
	if err != nil {
		return nil, err
	}

	v := &Job{}
	v.ID = v0
	v.BuildID = v1
	v.Number = v2
	v.Status = v3
	v.ExitCode = v4
	v.Started = v5
	v.Finished = v6
	json.Unmarshal(v7, &v.Environment)

	return v, nil
}

func scanJobs(rows *sql.Rows) ([]*Job, error) {
	var err error
	var vv []*Job
	for rows.Next() {
		var v0 int64
		var v1 int64
		var v2 int
		var v3 string
		var v4 int
		var v5 int64
		var v6 int64
		var v7 []byte
		err = rows.Scan(
			&v0,
			&v1,
			&v2,
			&v3,
			&v4,
			&v5,
			&v6,
			&v7,
		)
		if err != nil {
			return vv, err
		}

		v := &Job{}
		v.ID = v0
		v.BuildID = v1
		v.Number = v2
		v.Status = v3
		v.ExitCode = v4
		v.Started = v5
		v.Finished = v6
		json.Unmarshal(v7, &v.Environment)
		vv = append(vv, v)
	}
	return vv, rows.Err()
}

const stmtJobSelectList = `
SELECT
 job_id
,job_build_id
,job_number
,job_status
,job_exit_code
,job_started
,job_finished
,job_environment
FROM jobs
`

const stmtJobSelectRange = `
SELECT
 job_id
,job_build_id
,job_number
,job_status
,job_exit_code
,job_started
,job_finished
,job_environment
FROM jobs
LIMIT ? OFFSET ?
`

const stmtJobSelect = `
SELECT
 job_id
,job_build_id
,job_number
,job_status
,job_exit_code
,job_started
,job_finished
,job_environment
FROM jobs
WHERE job_id = ?
`

const stmtJobSelectJobBuildId = `
SELECT
 job_id
,job_build_id
,job_number
,job_status
,job_exit_code
,job_started
,job_finished
,job_environment
FROM jobs
WHERE job_build_id = ?
`

const stmtJobSelectBuildNumber = `
SELECT
 job_id
,job_build_id
,job_number
,job_status
,job_exit_code
,job_started
,job_finished
,job_environment
FROM jobs
WHERE job_build_id = ?
AND job_number = ?
`

const stmtJobSelectCount = `
SELECT count(1)
FROM jobs
`

const stmtJobInsert = `
INSERT INTO jobs (
 job_build_id
,job_number
,job_status
,job_exit_code
,job_started
,job_finished
,job_environment
) VALUES (?,?,?,?,?,?,?);
`

const stmtJobUpdate = `
UPDATE jobs SET
 job_build_id = ?
,job_number = ?
,job_status = ?
,job_exit_code = ?
,job_started = ?
,job_finished = ?
,job_environment = ?
WHERE job_id = ?
`

const stmtJobDelete = `
DELETE FROM jobs
WHERE job_id = ?
`

const stmtJobTable = `
CREATE TABLE IF NOT EXISTS jobs (
 job_id		INTEGER PRIMARY KEY AUTOINCREMENT
,job_build_id	INTEGER
,job_number	INTEGER
,job_status	VARCHAR
,job_exit_code	INTEGER
,job_started	INTEGER
,job_finished	INTEGER
,job_environmentVARCHAR(2048)
);
`

const stmtJobJobBuildIdIndex = `
CREATE INDEX IF NOT EXISTS ix_job_build_id ON jobs (job_build_id);
`

const stmtJobBuildNumberIndex = `
CREATE UNIQUE INDEX IF NOT EXISTS ux_build_number ON jobs (job_build_id,job_number);
`
