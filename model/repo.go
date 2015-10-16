package model

import (
	"github.com/drone/drone/shared/database"
	"github.com/russross/meddler"
	"strings"
)

type RepoLite struct {
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Avatar   string `json:"avatar_url"`
}

type Repo struct {
	ID          int64  `json:"id"                meddler:"repo_id,pk"`
	UserID      int64  `json:"-"                 meddler:"repo_user_id"`
	Owner       string `json:"owner"             meddler:"repo_owner"`
	Name        string `json:"name"              meddler:"repo_name"`
	FullName    string `json:"full_name"         meddler:"repo_full_name"`
	Avatar      string `json:"avatar_url"        meddler:"repo_avatar"`
	Link        string `json:"link_url"          meddler:"repo_link"`
	Clone       string `json:"clone_url"         meddler:"repo_clone"`
	Branch      string `json:"default_branch"    meddler:"repo_branch"`
	Timeout     int64  `json:"timeout"           meddler:"repo_timeout"`
	IsPrivate   bool   `json:"private"           meddler:"repo_private"`
	IsTrusted   bool   `json:"trusted"           meddler:"repo_trusted"`
	IsStarred   bool   `json:"starred,omitempty" meddler:"-"`
	AllowPull   bool   `json:"allow_pr"          meddler:"repo_allow_pr"`
	AllowPush   bool   `json:"allow_push"        meddler:"repo_allow_push"`
	AllowDeploy bool   `json:"allow_deploys"     meddler:"repo_allow_deploys"`
	AllowTag    bool   `json:"allow_tags"        meddler:"repo_allow_tags"`
	Hash        string `json:"-"                 meddler:"repo_hash"`
}

func GetRepo(db meddler.DB, id int64) (*Repo, error) {
	var repo = new(Repo)
	var err = meddler.Load(db, repoTable, repo, id)
	return repo, err
}

func GetRepoName(db meddler.DB, owner, name string) (*Repo, error) {
	return GetRepoFullName(db, owner+"/"+name)
}

func GetRepoFullName(db meddler.DB, name string) (*Repo, error) {
	var repo = new(Repo)
	var err = meddler.QueryRow(db, repo, database.Rebind(repoNameQuery), name)
	return repo, err
}

func GetRepoList(db meddler.DB, user *User) ([]*Repo, error) {
	// we don't have real-time access to the intersection
	// of github repos and drone repos. So we cheat by simply
	// getting the distinct list of repos that the user
	// has created builds for.
	var repos = []*Repo{}
	var err = meddler.QueryAll(db, &repos, database.Rebind(repoListQuery), user.Login)
	return repos, err
}

func GetRepoListOf(db meddler.DB, listof []*RepoLite) ([]*Repo, error) {
	var repos = []*Repo{}
	var size = len(listof)
	var qs = make([]string, size, size)
	var in = make([]interface{}, size, size)
	for i, repo := range listof {
		qs[i] = "?"
		in[i] = repo.FullName
	}
	var stmt = "SELECT * FROM repos WHERE repo_full_name IN (" + strings.Join(qs, ",") + ") ORDER BY repo_name"
	var err = meddler.QueryAll(db, &repos, database.Rebind(stmt), in...)
	return repos, err
}

func CreateRepo(db meddler.DB, repo *Repo) error {
	return meddler.Insert(db, repoTable, repo)
}

func UpdateRepo(db meddler.DB, repo *Repo) error {
	return meddler.Update(db, repoTable, repo)
}

func DeleteRepo(db meddler.DB, repo *Repo) error {
	var _, err = db.Exec(database.Rebind(repoDeleteStmt), repo.ID)
	return err
}

const repoTable = "repos"

const repoNameQuery = `
SELECT *
FROM repos
WHERE repo_full_name = ?
LIMIT 1;
`

const repoStarsQuery = `
SELECT r.*
FROM
 repos r
,stars s
WHERE r.repo_id = s.star_repo_id
  AND s.star_user_id = ?
`

const repoListQuery = `
SELECT *
FROM repos
WHERE repo_id IN (
	SELECT DISTINCT build_repo_id
	FROM builds
	WHERE build_author = ?
)
ORDER BY repo_full_name
`

const repoDeleteStmt = `
DELETE FROM repos
WHERE repo_id = ?
`
