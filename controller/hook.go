package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/engine"
	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/context"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/yaml"
	"github.com/drone/drone/yaml/matrix"
)

func PostHook(c *gin.Context) {
	remote := context.Remote(c)
	db := context.Database(c)

	tmprepo, build, err := remote.Hook(c.Request)
	if err != nil {
		log.Errorf("failure to parse hook. %s", err)
		c.AbortWithError(400, err)
		return
	}
	if build == nil {
		c.Writer.WriteHeader(200)
		return
	}
	if tmprepo == nil {
		log.Errorf("failure to ascertain repo from hook.")
		c.Writer.WriteHeader(400)
		return
	}

	// a build may be skipped if the text [CI SKIP]
	// is found inside the commit message
	if strings.Contains(build.Message, "[CI SKIP]") {
		log.Infof("ignoring hook. [ci skip] found for %s")
		c.Writer.WriteHeader(204)
		return
	}

	repo, err := model.GetRepoName(db, tmprepo.Owner, tmprepo.Name)
	if err != nil {
		log.Errorf("failure to find repo %s/%s from hook. %s", tmprepo.Owner, tmprepo.Name, err)
		c.AbortWithError(404, err)
		return
	}

	// get the token and verify the hook is authorized
	parsed, err := token.ParseRequest(c.Request, func(t *token.Token) (string, error) {
		return repo.Hash, nil
	})
	if err != nil {
		log.Errorf("failure to parse token from hook for %s. %s", repo.FullName, err)
		c.AbortWithError(400, err)
		return
	}
	if parsed.Text != repo.FullName {
		log.Errorf("failure to verify token from hook. Expected %s, got %s", repo.FullName, parsed.Text)
		c.AbortWithStatus(403)
		return
	}

	if repo.UserID == 0 {
		log.Warnf("ignoring hook. repo %s has no owner.", repo.FullName)
		c.Writer.WriteHeader(204)
		return
	}
	var skipped = true
	if (build.Event == model.EventPush && repo.AllowPush) ||
		(build.Event == model.EventPull && repo.AllowPull) ||
		(build.Event == model.EventDeploy && repo.AllowDeploy) ||
		(build.Event == model.EventTag && repo.AllowTag) {
		skipped = false
	}

	if skipped {
		log.Infof("ignoring hook. repo %s is disabled for %s events.", repo.FullName, build.Event)
		c.Writer.WriteHeader(204)
		return
	}

	user, err := model.GetUser(db, repo.UserID)
	if err != nil {
		log.Errorf("failure to find repo owner %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	// fetch the .drone.yml file from the database
	raw, sec, err := remote.Script(user, repo, build)
	if err != nil {
		log.Errorf("failure to get .drone.yml for %s. %s", repo.FullName, err)
		c.AbortWithError(404, err)
		return
	}

	axes, err := matrix.Parse(string(raw))
	if err != nil {
		log.Errorf("failure to calculate matrix for %s. %s", repo.FullName, err)
		c.AbortWithError(400, err)
		return
	}
	if len(axes) == 0 {
		axes = append(axes, matrix.Axis{})
	}

	netrc, err := remote.Netrc(user, repo)
	if err != nil {
		log.Errorf("failure to generate netrc for %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	key, _ := model.GetKey(db, repo)

	// verify the branches can be built vs skipped
	yconfig, _ := yaml.Parse(string(raw))
	var match = false
	for _, branch := range yconfig.Branches {
		if branch == build.Branch {
			match = true
			break
		}
		match, _ = filepath.Match(branch, build.Branch)
		if match {
			break
		}
	}
	if !match && len(yconfig.Branches) != 0 {
		log.Infof("ignoring hook. yaml file excludes repo and branch %s %s", repo.FullName, build.Branch)
		c.AbortWithStatus(200)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		log.Errorf("failure to begin database transaction", err)
		c.AbortWithError(500, err)
		return
	}
	defer tx.Rollback()

	// update some build fields
	build.Status = model.StatusPending
	build.RepoID = repo.ID

	var jobs []*model.Job
	for num, axis := range axes {
		jobs = append(jobs, &model.Job{
			BuildID:     build.ID,
			Number:      num + 1,
			Status:      model.StatusPending,
			Environment: axis,
		})
	}
	err = model.CreateBuild(tx, build, jobs...)
	if err != nil {
		log.Errorf("failure to save commit for %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}
	tx.Commit()

	c.JSON(200, build)

	url := fmt.Sprintf("%s/%s/%d", httputil.GetURL(c.Request), repo.FullName, build.Number)
	err = remote.Status(user, repo, build, url)
	if err != nil {
		log.Errorf("error setting commit status for %s/%d", repo.FullName, build.Number)
	}

	// get the previous build so taht we can send
	// on status change notifications
	last, _ := model.GetBuildLast(db, repo, build.Branch)

	engine_ := context.Engine(c)
	go engine_.Schedule(&engine.Task{
		User:      user,
		Repo:      repo,
		Build:     build,
		BuildPrev: last,
		Jobs:      jobs,
		Keys:      key,
		Netrc:     netrc,
		Config:    string(raw),
		Secret:    string(sec),
		System: &model.System{
			Link:    httputil.GetURL(c.Request),
			Plugins: strings.Split(os.Getenv("PLUGIN_FILTER"), " "),
			Globals: strings.Split(os.Getenv("PLUGIN_PARAMS"), " "),
		},
	})

}
