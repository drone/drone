package queue

import (
	"bytes"
	"fmt"
	"github.com/drone/drone/server/channel"
	"github.com/drone/drone/shared/build/git"
	r "github.com/drone/drone/shared/build/repo"
	//"github.com/drone/drone/pkg/plugin/notify"
	"io"
	"path/filepath"
	"time"

	"github.com/drone/drone/server/resource/build"
	"github.com/drone/drone/server/resource/commit"
	"github.com/drone/drone/server/resource/repo"
)

type worker struct {
	builds  build.BuildManager
	commits commit.CommitManager

	runner BuildRunner
}

// work is a function that will infinitely
// run in the background waiting for tasks that
// it can pull off the queue and execute.
func (w *worker) work(queue <-chan *BuildTask) {
	var task *BuildTask
	for {
		// get work item (pointer) from the queue
		task = <-queue
		if task == nil {
			continue
		}

		// execute the task
		w.execute(task)
	}
}

// execute will execute the build task and persist
// the results to the datastore.
func (w *worker) execute(task *BuildTask) error {
	// we need to be sure that we can recover
	// from any sort panic that could occur
	// to avoid brining down the entire application
	defer func() {
		if e := recover(); e != nil {
			task.Build.Finished = time.Now().Unix()
			task.Commit.Finished = time.Now().Unix()
			task.Build.Duration = task.Build.Finished - task.Build.Started
			task.Commit.Duration = task.Build.Finished - task.Build.Started
			task.Commit.Status = "Error"
			task.Build.Status = "Error"
			w.builds.Update(task.Build)
			w.commits.Update(task.Commit)
		}
	}()

	// update commit and build status
	task.Commit.Status = "Started"
	task.Build.Status = "Started"
	task.Build.Started = time.Now().Unix()
	task.Commit.Started = time.Now().Unix()

	// persist the commit to the database
	if err := w.commits.Update(task.Commit); err != nil {
		return err
	}

	// persist the build to the database
	if err := w.builds.Update(task.Build); err != nil {
		return err
	}

	// get settings
	//settings, _ := database.GetSettings()

	// notification context
	//context := &notify.Context{
	//	Repo:   task.Repo,
	//	Commit: task.Commit,
	//	Host:   settings.URL().String(),
	//}

	// send all "started" notifications
	//if task.Script.Notifications != nil {
	//	task.Script.Notifications.Send(context)
	//}

	// Send "started" notification to Github
	//if err := updateGitHubStatus(task.Repo, task.Commit); err != nil {
	//	log.Printf("error updating github status: %s\n", err.Error())
	//}

	// make sure a channel exists for the repository,
	// the commit, and the commit output (TODO)
	reposlug := fmt.Sprintf("%s/%s/%s", task.Repo.Remote, task.Repo.Owner, task.Repo.Name)
	commitslug := fmt.Sprintf("%s/%s/%s/commit/%s/%s", task.Repo.Remote, task.Repo.Owner, task.Repo.Name, task.Commit.Branch, task.Commit.Sha)
	consoleslug := fmt.Sprintf("%s/%s/%s/commit/%s/%s/builds/%d", task.Repo.Remote, task.Repo.Owner, task.Repo.Name, task.Commit.Branch, task.Commit.Sha, task.Build.Number)
	channel.Create(reposlug)
	channel.Create(commitslug)
	channel.CreateStream(consoleslug)

	// notify the channels that the commit and build started
	channel.SendJSON(reposlug, task.Commit)
	channel.SendJSON(commitslug, task.Build)

	var buf = &bufferWrapper{channel: consoleslug}

	// append private parameters to the environment
	// variable section of the .drone.yml file, iff
	// this is not a pull request (for security purposes)
	//if task.Repo.Params != nil && len(task.Commit.PullRequest) == 0 {
	//	for k, v := range task.Repo.Params {
	//		task.Script.Env = append(task.Script.Env, k+"="+v)
	//	}
	//}

	//defer func() {
	//	// update the status of the commit using the
	//	// GitHub status API.
	//	if err := updateGitHubStatus(task.Repo, task.Commit); err != nil {
	//		log.Printf("error updating github status: %s\n", err.Error())
	//	}
	//}()

	// execute the build
	passed, buildErr := w.runBuild(task, buf)

	task.Build.Finished = time.Now().Unix()
	task.Commit.Finished = time.Now().Unix()
	task.Build.Duration = task.Build.Finished - task.Build.Started
	task.Commit.Duration = task.Build.Finished - task.Build.Started
	task.Commit.Status = "Success"
	task.Build.Status = "Success"

	// capture build output
	stdout := buf.buf.String()

	// if exit code != 0 set to failure
	if passed {
		task.Commit.Status = "Failure"
		task.Build.Status = "Failure"
		if buildErr != nil && len(stdout) == 0 {
			// TODO: If you wanted to have very friendly error messages, you could do that here
			stdout = fmt.Sprintf("%s\n", buildErr.Error())
		}
	}

	// persist the build to the database
	if err := w.builds.Update(task.Build); err != nil {
		return err
	}

	// persist the build output
	if err := w.builds.UpdateOutput(task.Build, []byte(stdout)); err != nil {
		return nil
	}

	// persist the commit to the database
	if err := w.commits.Update(task.Commit); err != nil {
		return err
	}

	// notify the channels that the commit and build finished
	channel.SendJSON(reposlug, task.Commit)
	channel.SendJSON(commitslug, task.Build)
	channel.Close(consoleslug)

	// send all "finished" notifications
	//if task.Script.Notifications != nil {
	//	task.Script.Notifications.Send(context)
	//}

	return nil
}

func (w *worker) runBuild(task *BuildTask, buf io.Writer) (bool, error) {
	repo := &r.Repo{
		Name:   task.Repo.FullName,
		Path:   task.Repo.URL,
		Branch: task.Commit.Branch,
		Commit: task.Commit.Sha,
		PR:     task.Commit.PullRequest,
		//TODO the builder should handle this
		Dir:   filepath.Join("/var/cache/drone/src", task.Repo.FullName),
		Depth: git.GitDepth(task.Script.Git),
	}

	return w.runner.Run(
		task.Script,
		repo,
		[]byte(task.Repo.PrivateKey),
		task.Repo.Privileged,
		buf,
	)
}

// updateGitHubStatus is a helper function that will send
// the build status to GitHub using the Status API.
// see https://github.com/blog/1227-commit-status-api
func updateGitHubStatus(repo *repo.Repo, commit *commit.Commit) error {
	/*
		// convert from drone status to github status
		var message, status string
		switch commit.Status {
		case "Success":
			status = "success"
			message = "The build succeeded on drone.io"
		case "Failure":
			status = "failure"
			message = "The build failed on drone.io"
		case "Started":
			status = "pending"
			message = "The build is pending on drone.io"
		default:
			status = "error"
			message = "The build errored on drone.io"
		}

		// get the system settings
		settings, _ := database.GetSettings()

		// get the user from the database
		// since we need his / her GitHub token
		user, err := database.GetUser(repo.UserID)
		if err != nil {
			return err
		}

		client := github.New(user.GithubToken)
		client.ApiUrl = settings.GitHubApiUrl
		buildUrl := getBuildUrl(settings.URL().String(), repo, commit)

		return client.Repos.CreateStatus(repo.Owner, repo.Name, status, buildUrl, message, commit.Hash)
	*/
	return nil
}

/*
func getBuildUrl(host string, repo *repo.Repo, commit *commit.Commit) string {
	branchQuery := url.Values{}
	branchQuery.Set("branch", commit.Branch)
	buildUrl := fmt.Sprintf("%s/%s/commit/%s?%s", host, repo.Slug, commit.Hash, branchQuery.Encode())
	return buildUrl
}
*/

type bufferWrapper struct {
	buf bytes.Buffer

	// name of the channel
	channel string
}

func (b *bufferWrapper) Write(p []byte) (n int, err error) {
	n, err = b.buf.Write(p)
	channel.SendBytes(b.channel, p)
	return
}
