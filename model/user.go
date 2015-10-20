package model

import (
	"github.com/drone/drone/shared/database"
	"github.com/russross/meddler"
)

type User struct {
	ID     int64  `json:"id"         meddler:"user_id,pk"`
	Login  string `json:"login"      meddler:"user_login"`
	Token  string `json:"-"          meddler:"user_token"`
	Secret string `json:"-"          meddler:"user_secret"`
	Expiry int64  `json:"-"          meddler:"user_expiry"`
	Email  string `json:"email"      meddler:"user_email"`
	Avatar string `json:"avatar_url" meddler:"user_avatar"`
	Active bool   `json:"active,"    meddler:"user_active"`
	Admin  bool   `json:"admin,"     meddler:"user_admin"`
	Hash   string `json:"-"          meddler:"user_hash"`
}

func GetUser(db meddler.DB, id int64) (*User, error) {
	var usr = new(User)
	var err = meddler.Load(db, userTable, usr, id)
	return usr, err
}

func GetUserLogin(db meddler.DB, login string) (*User, error) {
	var usr = new(User)
	var err = meddler.QueryRow(db, usr, database.Rebind(userLoginQuery), login)
	return usr, err
}

func GetUserList(db meddler.DB) ([]*User, error) {
	var users = []*User{}
	var err = meddler.QueryAll(db, &users, database.Rebind(userListQuery))
	return users, err
}

func GetUserFeed(db meddler.DB, user *User, limit, offset int) ([]*Feed, error) {
	var feed = []*Feed{}
	var err = meddler.QueryAll(db, &feed, database.Rebind(userFeedQuery), user.Login, limit, offset)
	return feed, err
}

func GetUserCount(db meddler.DB) (int, error) {
	var count int
	var err = db.QueryRow(database.Rebind(userCountQuery)).Scan(&count)
	return count, err
}

func CreateUser(db meddler.DB, user *User) error {
	return meddler.Insert(db, userTable, user)
}

func UpdateUser(db meddler.DB, user *User) error {
	return meddler.Update(db, userTable, user)
}

func DeleteUser(db meddler.DB, user *User) error {
	var _, err = db.Exec(database.Rebind(userDeleteStmt), user.ID)
	return err
}

const userTable = "users"

const userLoginQuery = `
SELECT *
FROM users
WHERE user_login=?
LIMIT 1
`

const userListQuery = `
SELECT *
FROM users
ORDER BY user_login ASC
`

const userCountQuery = `
SELECT count(1)
FROM users
`

const userDeleteStmt = `
DELETE FROM users
WHERE user_id=?
`

const userFeedQuery = `
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
  AND b.build_author = ?
ORDER BY b.build_id DESC
LIMIT ? OFFSET ?
`
