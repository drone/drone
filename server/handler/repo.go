package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/resource/config"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/sshutil"
	"github.com/gorilla/pat"
)

type RepoHandler struct {
	conf    *config.Config
	commits database.CommitManager
	perms   database.PermManager
	repos   database.RepoManager
	sess    session.Session
}

func NewRepoHandler(repos database.RepoManager, commits database.CommitManager,
	perms database.PermManager, sess session.Session, conf *config.Config) *RepoHandler {
	return &RepoHandler{conf, commits, perms, repos, sess}
}

// GetRepo gets the named repository.
// GET /v1/repos/:host/:owner/:name
func (h *RepoHandler) GetRepo(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)

	// get the user form the session.
	user := h.sess.User(r)

	// get the repository from the database.
	repo, err := h.repos.FindName(host, owner, name)
	if err != nil {
		return notFound{err}
	}

	// user must have read access to the repository.
	if ok, _ := h.perms.Read(user, repo); !ok {
		return notFound{err}
	}

	return json.NewEncoder(w).Encode(repo)
}

// PostRepo activates the named repository.
// POST /v1/repos/:host/:owner/:name
func (h *RepoHandler) PostRepo(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)

	// get the user form the session.
	user := h.sess.User(r)
	if user == nil {
		return notAuthorized{}
	}

	// get the repo from the database
	repo, err := h.repos.FindName(host, owner, name)
	if err != nil {
		return notFound{err}
	}

	// user must have admin access to the repository.
	if ok, _ := h.perms.Admin(user, repo); !ok {
		return notFound{err}
	}

	// update the repo active flag and fields
	repo.Active = true
	repo.PullRequest = true
	repo.PostCommit = true
	repo.UserID = user.ID

	// generate the rsa key
	key, err := sshutil.GeneratePrivateKey()
	if err != nil {
		return internalServerError{err}
	}

	// marshal the public and private key values
	repo.PublicKey = sshutil.MarshalPublicKey(&key.PublicKey)
	repo.PrivateKey = sshutil.MarshalPrivateKey(key)

	// get the remote and client
	remote := h.conf.GetRemote(host)
	if remote == nil {
		return notFound{}
	}

	// post commit hook url
	hook := fmt.Sprintf("%s://%s/hook/%s", httputil.GetScheme(r), httputil.GetHost(r), remote.GetName())

	// activate the repository in the remote system
	client := remote.GetClient(user.Access, user.Secret)
	if err := client.SetActive(owner, name, hook, repo.PublicKey); err != nil {
		return badRequest{err}
	}

	// update the status in the database
	if err := h.repos.Update(repo); err != nil {
		return badRequest{err}
	}

	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(repo)
}

// PutRepo updates the named repository.
// PUT /v1/repos/:host/:owner/:name
func (h *RepoHandler) PutRepo(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)

	// get the user form the session.
	user := h.sess.User(r)
	if user == nil {
		return notAuthorized{}
	}

	// get the repo from the database
	repo, err := h.repos.FindName(host, owner, name)
	if err != nil {
		return notFound{err}
	}

	// user must have admin access to the repository.
	if ok, _ := h.perms.Admin(user, repo); !ok {
		return notFound{err}
	}

	// unmarshal the repository from the payload
	defer r.Body.Close()
	in := struct {
		PostCommit  *bool   `json:"post_commits"`
		PullRequest *bool   `json:"pull_requests"`
		Privileged  *bool   `json:"privileged"`
		Params      *string `json:"params"`
		Timeout     *int64  `json:"timeout"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return badRequest{err}
	}

	// update the private/secure parameters
	if in.Params != nil {
		repo.Params = *in.Params
	}
	// update the post commit flag
	if in.PostCommit != nil {
		repo.PostCommit = *in.PostCommit
	}
	// update the pull request flag
	if in.PullRequest != nil {
		repo.PullRequest = *in.PullRequest
	}
	// update the privileged flag. This can only be updated by
	// the system administrator
	if in.Privileged != nil && user.Admin {
		repo.Privileged = *in.Privileged
	}
	// update the timeout. This can only be updated by
	// the system administrator
	if in.Timeout != nil && user.Admin {
		repo.Timeout = *in.Timeout
	}

	// update the repository
	if err := h.repos.Update(repo); err != nil {
		return badRequest{err}
	}

	return json.NewEncoder(w).Encode(repo)
}

// DeleteRepo deletes the named repository.
// DEL /v1/repos/:host/:owner/:name
func (h *RepoHandler) DeleteRepo(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)

	// get the user form the session.
	user := h.sess.User(r)
	if user == nil {
		return notAuthorized{}
	}

	// get the repo from the database
	repo, err := h.repos.FindName(host, owner, name)
	if err != nil {
		return notFound{err}
	}

	// user must have admin access to the repository.
	if ok, _ := h.perms.Admin(user, repo); !ok {
		return notFound{err}
	}

	// update the repo active flag and fields.
	repo.Active = false
	repo.PullRequest = false
	repo.PostCommit = false

	// insert the new repository
	if err := h.repos.Update(repo); err != nil {
		return badRequest{err}
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// GetFeed gets the most recent commits across all branches
// GET /v1/repos/{host}/{owner}/{name}/feed
func (h *RepoHandler) GetFeed(w http.ResponseWriter, r *http.Request) error {
	var host, owner, name = parseRepo(r)

	// get the user form the session.
	user := h.sess.User(r)

	// get the repository from the database.
	repo, err := h.repos.FindName(host, owner, name)
	if err != nil {
		return notFound{err}
	}

	// user must have read access to the repository.
	if ok, _ := h.perms.Read(user, repo); !ok {
		return notFound{err}
	}

	// lists the most recent commits across all branches.
	commits, err := h.commits.List(repo.ID)
	if err != nil {
		return notFound{err}
	}

	return json.NewEncoder(w).Encode(commits)
}

func (h *RepoHandler) Register(r *pat.Router) {
	r.Get("/v1/repos/{host}/{owner}/{name}/feed", errorHandler(h.GetFeed))
	r.Get("/v1/repos/{host}/{owner}/{name}", errorHandler(h.GetRepo))
	r.Put("/v1/repos/{host}/{owner}/{name}", errorHandler(h.PutRepo))
	r.Post("/v1/repos/{host}/{owner}/{name}", errorHandler(h.PostRepo))
	r.Delete("/v1/repos/{host}/{owner}/{name}", errorHandler(h.DeleteRepo))
}
