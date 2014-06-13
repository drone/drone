package handler

import (
	"net/http"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/queue"
	"github.com/drone/drone/shared/model"
	"github.com/gorilla/pat"
)

type HookHandler struct {
	users   database.UserManager
	repos   database.RepoManager
	commits database.CommitManager
	conf    database.ConfigManager
	queue   *queue.Queue
}

func NewHookHandler(users database.UserManager, repos database.RepoManager, commits database.CommitManager, conf database.ConfigManager, queue *queue.Queue) *HookHandler {
	return &HookHandler{users, repos, commits, conf, queue}
}

// PostHook receives a post-commit hook from GitHub, Bitbucket, etc
// GET /hook/:host
func (h *HookHandler) PostHook(w http.ResponseWriter, r *http.Request) error {
	host := r.FormValue(":host")

	// get the remote system's client.
	remote := h.conf.Find().GetRemote(host)
	if remote == nil {
		return notFound{}
	}

	// parse the hook payload
	hook, err := remote.GetHook(r)
	if err != nil {
		return badRequest{err}
	}

	// in some cases we have neither a hook nor error. An example
	// would be GitHub sending a ping request to the URL, in which
	// case we'll just exit quiely with an 'OK'
	if hook == nil {
		w.WriteHeader(http.StatusOK)
		return nil
	}

	//fmt.Printf("%#v", hook)

	// fetch the repository from the database
	repo, err := h.repos.FindName(remote.GetHost(), hook.Owner, hook.Repo)
	if err != nil {
		return notFound{}
	}

	if repo.Active == false ||
		(repo.PostCommit == false && len(hook.PullRequest) == 0) ||
		(repo.PullRequest == false && len(hook.PullRequest) != 0) {
		w.WriteHeader(http.StatusOK)
		return nil
	}

	// fetch the user from the database that owns this repo
	user, err := h.users.Find(repo.UserID)
	if err != nil {
		return notFound{}
	}

	// featch the .drone.yml file from the database
	client := remote.GetClient(user.Access, user.Secret)
	yml, err := client.GetScript(hook)
	if err != nil {
		return badRequest{err}
	}

	c := model.Commit{
		RepoID:      repo.ID,
		Status:      model.StatusEnqueue,
		Sha:         hook.Sha,
		Branch:      hook.Branch,
		PullRequest: hook.PullRequest,
		Timestamp:   hook.Timestamp,
		Message:     hook.Message,
		Config:      yml}
	c.SetAuthor(hook.Author)
	// inser the commit into the database
	if err := h.commits.Insert(&c); err != nil {
		return badRequest{err}
	}

	//fmt.Printf("%s", yml)

	// drop the items on the queue
	h.queue.Add(&queue.BuildTask{Repo: repo, Commit: &c})
	return nil
}

func (h *HookHandler) Register(r *pat.Router) {
	r.Post("/hook/{host}", errorHandler(h.PostHook))
	r.Put("/hook/{host}", errorHandler(h.PostHook))
}
