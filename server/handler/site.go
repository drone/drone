package handler

import (
	"net/http"

	"github.com/drone/drone/server/render"
	"github.com/drone/drone/server/resource/build"
	"github.com/drone/drone/server/resource/commit"
	"github.com/drone/drone/server/resource/perm"
	"github.com/drone/drone/server/resource/repo"
	"github.com/drone/drone/server/resource/user"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/shared/util/httputil"
	"github.com/gorilla/pat"
)

type SiteHandler struct {
	users   user.UserManager
	repos   repo.RepoManager
	commits commit.CommitManager
	builds  build.BuildManager
	perms   perm.PermManager
	sess    session.Session
	render  render.Render
}

func NewSiteHandler(users user.UserManager,
	repos repo.RepoManager, commits commit.CommitManager, builds build.BuildManager,
	perms perm.PermManager, sess session.Session, render render.Render) *SiteHandler {
	return &SiteHandler{users, repos, commits, builds, perms, sess, render}
}

// GetIndex serves the root domain request. This is forwarded to the dashboard
// page iff the user is authenticated, else it is forwarded to the login page.
func (s *SiteHandler) GetIndex(w http.ResponseWriter, r *http.Request) error {
	u := s.sess.User(r)
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil
	}
	feed, _ := s.commits.ListUser(u.ID)
	data := struct {
		User *user.User
		Feed []*commit.CommitRepo
	}{u, feed}
	return s.render(w, "user_feed.html", &data)
	/*
		u := s.sess.User(r)
		data := struct {
			User *user.User
		}{u}
		return s.render(w, "index.html", &data)
	*/
}

func (s *SiteHandler) GetUser(w http.ResponseWriter, r *http.Request) error {
	u := s.sess.User(r)
	if u == nil {
		return s.render(w, "404.html", nil)
	}
	return s.render(w, "user_conf.html", struct{ User *user.User }{u})
}

func (s *SiteHandler) GetUsers(w http.ResponseWriter, r *http.Request) error {
	u := s.sess.User(r)
	if u == nil || u.Admin == false {
		return s.render(w, "404.html", nil)
	}
	return s.render(w, "admin_users.html", struct{ User *user.User }{u})
}

func (s *SiteHandler) GetConfig(w http.ResponseWriter, r *http.Request) error {
	u := s.sess.User(r)
	if u == nil || u.Admin == false {
		return s.render(w, "404.html", nil)
	}
	return s.render(w, "admin_conf.html", struct{ User *user.User }{u})
}

func (s *SiteHandler) GetRepo(w http.ResponseWriter, r *http.Request) error {
	host, owner, name := parseRepo(r)
	branch := parseBranch(r)
	sha := parseCommit(r)
	usr := s.sess.User(r)

	arepo, err := s.repos.FindName(host, owner, name)
	if err != nil {
		return s.render(w, "404.html", nil)
	}
	if ok, _ := s.perms.Read(usr, arepo); !ok {
		return s.render(w, "404.html", nil)
	}
	data := struct {
		User     *user.User
		Repo     *repo.Repo
		Branch   string
		Token    string
		Branches []*commit.Commit
		Commits  []*commit.Commit
		Commit   *commit.Commit
		Builds   []*build.Build
		Build    *build.Build
	}{User: usr, Repo: arepo}

	// if commit details are provided we should retrieve the build details
	// and serve the build page.
	if len(sha) != 0 {
		data.Commit, err = s.commits.FindSha(data.Repo.ID, branch, sha)
		if err != nil {
			return s.render(w, "404.html", nil)
		}
		data.Build, err = s.builds.FindNumber(data.Commit.ID, 1)
		if err != nil {
			return s.render(w, "404.html", nil)
		}
		return s.render(w, "repo_commit.html", &data)
	}

	// retrieve the list of recently built branches
	data.Branches, _ = s.commits.ListBranches(data.Repo.ID)

	// if the branch parameter is provided we should retrieve the build
	// feed for this branch only.
	if len(branch) != 0 {
		data.Branch = branch
		data.Commits, err = s.commits.ListBranch(data.Repo.ID, branch)
		if err != nil {
			return s.render(w, "404.html", nil)
		}
		return s.render(w, "repo_branch.html", &data)
	}

	// else we should serve the standard build feed
	data.Commits, _ = s.commits.List(data.Repo.ID)
	return s.render(w, "repo_feed.html", &data)
}

func (s *SiteHandler) GetRepoAdmin(w http.ResponseWriter, r *http.Request) error {
	host, owner, name := parseRepo(r)
	arepo, err := s.repos.FindName(host, owner, name)
	u := s.sess.User(r)
	if err != nil {
		return s.render(w, "404.html", nil)
	}
	if ok, _ := s.perms.Admin(u, arepo); !ok {
		return s.render(w, "404.html", nil)
	}
	data := struct {
		User   *user.User
		Repo   *repo.Repo
		Host   string
		Scheme string
	}{u, arepo, httputil.GetHost(r), httputil.GetScheme(r)}
	return s.render(w, "repo_conf.html", &data)
}

func (s *SiteHandler) GetRepos(w http.ResponseWriter, r *http.Request) error {
	u := s.sess.User(r)
	if u == nil || u.Admin == false {
		return s.render(w, "404.html", nil)
	}
	repos, err := s.repos.List(u.ID)
	if err != nil {
		s.render(w, "500.html", nil)
	}
	data := struct {
		User  *user.User
		Repos []*repo.Repo
	}{u, repos}
	return s.render(w, "user_repos.html", &data)
}

func (s *SiteHandler) Register(r *pat.Router) {

	r.Get("/admin/users", errorHandler(s.GetUsers))
	r.Get("/admin/settings", errorHandler(s.GetConfig))
	r.Get("/account/profile", errorHandler(s.GetUser))
	r.Get("/account/repos", errorHandler(s.GetRepos))
	r.Get("/{host}/{owner}/{name}/settings", errorHandler(s.GetRepoAdmin))
	r.Get("/{host}/{owner}/{name}/branch/{branch}/commit/{commit}", errorHandler(s.GetRepo))
	r.Get("/{host}/{owner}/{name}/branch/{branch}", errorHandler(s.GetRepo))
	r.Get("/{host}/{owner}/{name}", errorHandler(s.GetRepo))
	r.Get("/", errorHandler(s.GetIndex))
}
