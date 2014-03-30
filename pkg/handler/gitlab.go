package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/drone/drone/pkg/build/script"
	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
	"github.com/drone/drone/pkg/queue"
	"github.com/plouc/go-gitlab-client"
)

type GitlabHandler struct {
	queue   *queue.Queue
	apiPath string
}

func NewGitlabHandler(queue *queue.Queue) *GitlabHandler {
	return &GitlabHandler{
		queue:   queue,
		apiPath: "/api/v3",
	}
}

func (g *GitlabHandler) Add(w http.ResponseWriter, r *http.Request, u *User) error {
	settings := database.SettingsMust()
	teams, err := database.ListTeams(u.ID)
	if err != nil {
		return err
	}
	data := struct {
		User     *User
		Teams    []*Team
		Settings *Settings
	}{u, teams, settings}
	// if the user hasn't linked their GitLab account
	// render a different template
	if len(u.GitlabToken) == 0 {
		return RenderTemplate(w, "gitlab_link.html", &data)
	}
	// otherwise display the template for adding
	// a new GitLab repository.
	return RenderTemplate(w, "gitlab_add.html", &data)
}

func (g *GitlabHandler) Link(w http.ResponseWriter, r *http.Request, u *User) error {
	token := r.FormValue("token")
	u.GitlabToken = token

	if err := database.SaveUser(u); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}

	settings := database.SettingsMust()
	gl := gogitlab.NewGitlab(settings.GitlabApiUrl, g.apiPath, u.GitlabToken)
	_, err := gl.CurrentUser()
	if err != nil {
		return fmt.Errorf("Private Token is not valid: %q", err)
	}

	http.Redirect(w, r, "/new/gitlab", http.StatusSeeOther)
	return nil
}

func (g *GitlabHandler) Create(w http.ResponseWriter, r *http.Request, u *User) error {
	teamName := r.FormValue("team")
	owner := r.FormValue("owner")
	name := r.FormValue("name")

	repo, err := g.newGitlabRepo(u, owner, name)
	if err != nil {
		return err
	}

	if len(teamName) > 0 {
		team, err := database.GetTeamSlug(teamName)
		if err != nil {
			return fmt.Errorf("Unable to find Team %s.", teamName)
		}

		// user must be an admin member of the team
		if ok, _ := database.IsMemberAdmin(u.ID, team.ID); !ok {
			return fmt.Errorf("Invalid permission to access Team %s.", teamName)
		}
		repo.TeamID = team.ID
	}

	// Save to the database
	if err := database.SaveRepo(repo); err != nil {
		return fmt.Errorf("Error saving repository to the database. %s", err)
	}

	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func (g *GitlabHandler) newGitlabRepo(u *User, owner, name string) (*Repo, error) {
	settings := database.SettingsMust()
	gl := gogitlab.NewGitlab(settings.GitlabApiUrl, g.apiPath, u.GitlabToken)

	project, err := gl.Project(ns(owner, name))
	if err != nil {
		return nil, err
	}

	var cloneUrl string
	if project.Public {
		cloneUrl = project.HttpRepoUrl
	} else {
		cloneUrl = project.SshRepoUrl
	}

	repo, err := NewRepo(settings.GitlabDomain, owner, name, ScmGit, cloneUrl)
	if err != nil {
		return nil, err
	}

	repo.UserID = u.ID
	repo.Private = !project.Public
	if repo.Private {
		// name the key
		keyName := fmt.Sprintf("%s@%s", repo.Owner, settings.Domain)

		// TODO: (fudanchii) check if we already opted to use UserKey

		// create the github key, or update if one already exists
		if err := gl.AddProjectDeployKey(ns(owner, name), keyName, repo.PublicKey); err != nil {
			return nil, fmt.Errorf("Unable to add Public Key to your GitLab repository.")
		}
	}

	link := fmt.Sprintf("%s://%s/hook/gitlab?id=%s", settings.Scheme, settings.Domain, repo.Slug)
	if err := gl.AddProjectHook(ns(owner, name), link, true, false, true); err != nil {
		return nil, fmt.Errorf("Unable to add Hook to your GitLab repository.")
	}

	return repo, err
}

func (g *GitlabHandler) Hook(w http.ResponseWriter, r *http.Request) error {
	var payload []byte
	n, err := r.Body.Read(payload)
	if n == 0 {
		return fmt.Errorf("Request Empty: %q", err)
	}
	parsed, err := gogitlab.ParseHook(payload)
	if err != nil {
		return err
	}
	if parsed.ObjectKind == "merge_request" {
		return g.PullRequestHook(parsed)
	}

	if len(parsed.After) == 0 {
		return RenderText(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	rID := r.FormValue("id")
	repo, err := database.GetRepoSlug(rID)
	if err != nil {
		return RenderText(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	user, err := database.GetUser(repo.UserID)
	if err != nil {
		return RenderText(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	_, err = database.GetCommitHash(parsed.After, repo.ID)
	if err != nil && err != sql.ErrNoRows {
		println("commit already exists")
		return RenderText(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
	}

	commit := &Commit{}
	commit.RepoID = repo.ID
	commit.Branch = parsed.Branch()
	commit.Hash = parsed.After
	commit.Status = "Pending"
	commit.Created = time.Now().UTC()

	head := parsed.Head()
	commit.Message = head.Message
	commit.Timestamp = head.Timestamp
	if head.Author != nil {
		commit.SetAuthor(head.Author.Email)
	} else {
		commit.SetAuthor(parsed.UserName)
	}

	// get the github settings from the database
	settings := database.SettingsMust()

	// get the drone.yml file from GitHub
	client := gogitlab.NewGitlab(settings.GitlabApiUrl, g.apiPath, user.GitlabToken)

	content, err := client.RepoRawFile(ns(repo.Owner, repo.Name), commit.Hash, ".drone.yml")
	if err != nil {
		msg := "No .drone.yml was found in this repository.  You need to add one.\n"
		if err := saveFailedBuild(commit, msg); err != nil {
			return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return RenderText(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}

	// parse the build script
	buildscript, err := script.ParseBuild(content, repo.Params)
	if err != nil {
		msg := "Could not parse your .drone.yml file.  It needs to be a valid drone yaml file.\n\n" + err.Error() + "\n"
		if err := saveFailedBuild(commit, msg); err != nil {
			return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return RenderText(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}

	// save the commit to the database
	if err := database.SaveCommit(commit); err != nil {
		return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	// save the build to the database
	build := &Build{}
	build.Slug = "1" // TODO
	build.CommitID = commit.ID
	build.Created = time.Now().UTC()
	build.Status = "Pending"
	if err := database.SaveBuild(build); err != nil {
		return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	// notify websocket that a new build is pending
	//realtime.CommitPending(repo.UserID, repo.TeamID, repo.ID, commit.ID, repo.Private)
	//realtime.BuildPending(repo.UserID, repo.TeamID, repo.ID, commit.ID, build.ID, repo.Private)

	g.queue.Add(&queue.BuildTask{Repo: repo, Commit: commit, Build: build, Script: buildscript}) //Push(repo, commit, build, buildscript)

	// OK!
	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)

}

func (g *GitlabHandler) PullRequestHook(p *gogitlab.HookPayload) error {
	return fmt.Errorf("Not implemented yet")
}

// ns namespaces user and repo.
// Returns user%2Frepo
func ns(user, repo string) string {
	return fmt.Sprintf("%s%%2F%s", user, repo)
}
