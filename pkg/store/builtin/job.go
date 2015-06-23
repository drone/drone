package builtin

import (
	"database/sql"

	"github.com/drone/drone/pkg/types"
)

type Jobstore struct {
	*sql.DB
}

func NewJobstore(db *sql.DB) *Jobstore {
	return &Jobstore{db}
}

// Job returns a Job by ID.
func (db *Jobstore) Job(id int64) (*types.Job, error) {
	return getJob(db, rebind(stmtJobSelect), id)
}

// JobNumber returns a job by sequence number.
func (db *Jobstore) JobNumber(build *types.Build, seq int) (*types.Job, error) {
	return getJob(db, rebind(stmtJobSelectBuildNumber), build.ID, seq)
}

// JobList returns a list of all build jobs
func (db *Jobstore) JobList(build *types.Build) ([]*types.Job, error) {
	return getJobs(db, rebind(stmtJobSelectJobBuildId), build.ID)
}

// SetJob updates an existing build job.
func (db *Jobstore) SetJob(job *types.Job) error {
	return updateJob(db, rebind(stmtJobUpdate), job)
}

// AddJob inserts a build job.
func (db *Jobstore) AddJob(job *types.Job) error {
	return createJob(db, rebind(stmtJobInsert), job)
}
