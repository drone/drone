package commit

import (
	"database/sql"
	"time"

	"github.com/russross/meddler"
)

type CommitManager interface {
	// Find finds the commit by ID.
	Find(id int64) (*Commit, error)

	// FindSha finds the commit for the branch and sha.
	FindSha(repo int64, branch, sha string) (*Commit, error)

	// FindLatest finds the most recent commit for the branch.
	FindLatest(repo int64, branch string) (*Commit, error)

	// FindOutput finds the commit's output.
	//FindOutput(id int64) ([]byte, error)

	// List finds recent commits for the repository
	List(repo int64) ([]*Commit, error)

	// ListBranch finds recent commits for the repository and branch.
	ListBranch(repo int64, branch string) ([]*Commit, error)

	// ListBranches finds most recent commit for each branch.
	ListBranches(repo int64) ([]*Commit, error)

	// ListUser finds most recent commits for a user.
	ListUser(repo int64) ([]*CommitRepo, error)

	// Insert persists the commit to the datastore.
	Insert(commit *Commit) error

	// Update persists changes to the commit to the datastore.
	Update(commit *Commit) error

	// UpdateOutput persists a commit's stdout to the datastore.
	//UpdateOutput(commit *Commit, out []byte) error

	// Delete removes the commit from the datastore.
	Delete(commit *Commit) error
}

// commitManager manages a list of commits in a SQL database.
type commitManager struct {
	*sql.DB
}

// NewManager initiales a new CommitManager intended to
// manage and persist commits.
func NewManager(db *sql.DB) CommitManager {
	return &commitManager{db}
}

// SQL query to retrieve the latest Commits for each branch.
const listBranchesQuery = `
SELECT *
FROM commits
WHERE commit_id IN (
    SELECT MAX(commit_id)
    FROM commits
    WHERE repo_id=?
    AND commit_status NOT IN ('Started', 'Pending')
    GROUP BY commit_branch)
 ORDER BY commit_branch ASC
 `

// SQL query to retrieve the latest Commits for a specific branch.
const listBranchQuery = `
SELECT *
FROM commits
WHERE repo_id=?
AND   commit_branch=?
ORDER BY commit_id DESC
LIMIT 20
 `

// SQL query to retrieve the latest Commits for a user's repositories.
const listUserQuery = `
SELECT r.repo_remote, r.repo_owner, r.repo_name, c.*
FROM commits c, repos r, perms p
WHERE c.repo_id=r.repo_id
AND   r.repo_id=p.repo_id
AND   p.user_id=?
AND   c.commit_status NOT IN ('Started', 'Pending')
ORDER BY commit_id DESC
LIMIT 20
`

// SQL query to retrieve the latest Commits across all branches.
const listQuery = `
SELECT *
FROM commits
WHERE repo_id=? 
ORDER BY commit_id DESC
LIMIT 20
`

// SQL query to retrieve a Commit by branch and sha.
const findQuery = `
SELECT *
FROM commits
WHERE repo_id=?
AND   commit_branch=?
AND   commit_sha=?
LIMIT 1
`

// SQL query to retrieve the most recent Commit for a branch.
const findLatestQuery = `
SELECT *
FROM commits
WHERE commit_id IN (
    SELECT MAX(commit_id)
    FROM commits
    WHERE repo_id=?
    AND commit_branch=?)
`

// SQL statement to delete a Commit by ID.
const deleteStmt = `
DELETE FROM commits WHERE commit_id = ?
`

func (db *commitManager) Find(id int64) (*Commit, error) {
	dst := Commit{}
	err := meddler.Load(db, "commits", &dst, id)
	return &dst, err
}

func (db *commitManager) FindSha(repo int64, branch, sha string) (*Commit, error) {
	dst := Commit{}
	err := meddler.QueryRow(db, &dst, findQuery, repo, branch, sha)
	return &dst, err
}

func (db *commitManager) FindLatest(repo int64, branch string) (*Commit, error) {
	dst := Commit{}
	err := meddler.QueryRow(db, &dst, findLatestQuery, repo, branch)
	return &dst, err
}

func (db *commitManager) List(repo int64) ([]*Commit, error) {
	var dst []*Commit
	err := meddler.QueryAll(db, &dst, listQuery, repo)
	return dst, err
}

func (db *commitManager) ListBranch(repo int64, branch string) ([]*Commit, error) {
	var dst []*Commit
	err := meddler.QueryAll(db, &dst, listBranchQuery, repo, branch)
	return dst, err
}

func (db *commitManager) ListBranches(repo int64) ([]*Commit, error) {
	var dst []*Commit
	err := meddler.QueryAll(db, &dst, listBranchesQuery, repo)
	return dst, err
}

func (db *commitManager) ListUser(user int64) ([]*CommitRepo, error) {
	var dst []*CommitRepo
	err := meddler.QueryAll(db, &dst, listUserQuery, user)
	return dst, err
}

func (db *commitManager) Insert(commit *Commit) error {
	commit.Created = time.Now().Unix()
	commit.Updated = time.Now().Unix()
	return meddler.Insert(db, "commits", commit)
}

func (db *commitManager) Update(commit *Commit) error {
	commit.Updated = time.Now().Unix()
	return meddler.Update(db, "commits", commit)
}

func (db *commitManager) Delete(commit *Commit) error {
	_, err := db.Exec(deleteStmt, commit.ID)
	return err
}
