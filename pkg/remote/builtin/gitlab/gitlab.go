package gitlab

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/Bugagazavr/go-gitlab-client"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/hashicorp/golang-lru"
	"github.com/drone/drone/pkg/oauth2"
	"github.com/drone/drone/pkg/remote"
	common "github.com/drone/drone/pkg/types"
	"github.com/drone/drone/pkg/utils/httputil"
)

const (
	DefaultScope = "api"
)

type Gitlab struct {
	URL         string
	Client      string
	Secret      string
	AllowedOrgs []string
	Open        bool
	PrivateMode bool
	SkipVerify  bool
	Search      bool

	cache *lru.Cache
}

func init() {
	remote.Register("gitlab", NewDriver)
}

func NewDriver(config string) (remote.Remote, error) {
	url_, err := url.Parse(config)
	if err != nil {
		return nil, err
	}
	params := url_.Query()
	url_.RawQuery = ""

	gitlab := Gitlab{}
	gitlab.URL = url_.String()
	gitlab.Client = params.Get("client_id")
	gitlab.Secret = params.Get("client_secret")
	gitlab.AllowedOrgs = params["orgs"]
	gitlab.SkipVerify, _ = strconv.ParseBool(params.Get("skip_verify"))
	gitlab.Open, _ = strconv.ParseBool(params.Get("open"))

	// this is a temp workaround
	gitlab.Search, _ = strconv.ParseBool(params.Get("search"))

	// here we cache permissions to avoid too many api
	// calls. this should really be moved outise the
	// remote plugin into the app
	gitlab.cache, err = lru.New(1028)
	return &gitlab, err
}

func (r *Gitlab) Login(token, secret string) (*common.User, error) {
	client := NewClient(r.URL, token, r.SkipVerify)
	var login, err = client.CurrentUser()
	if err != nil {
		return nil, err
	}
	user := common.User{}
	user.Login = login.Username
	user.Email = login.Email
	user.Token = token
	user.Secret = secret

	if strings.HasPrefix(login.AvatarUrl, "http") {
		user.Avatar = login.AvatarUrl
	} else {
		user.Avatar = r.URL + "/" + login.AvatarUrl
	}
	return &user, nil
}

// Orgs fetches the organizations for the given user.
func (r *Gitlab) Orgs(u *common.User) ([]string, error) {
	return nil, nil
}

// Repo fetches the named repository from the remote system.
func (r *Gitlab) Repo(u *common.User, owner, name string) (*common.Repo, error) {
	client := NewClient(r.URL, u.Token, r.SkipVerify)
	id, err := GetProjectId(r, client, owner, name)
	if err != nil {
		return nil, err
	}
	repo_, err := client.Project(id)
	if err != nil {
		return nil, err
	}

	repo := &common.Repo{}
	repo.Owner = owner
	repo.Name = name
	repo.FullName = repo_.PathWithNamespace
	repo.Link = repo_.Url
	repo.Clone = repo_.HttpRepoUrl
	repo.Branch = "master"

	if repo_.DefaultBranch != "" {
		repo.Branch = repo_.DefaultBranch
	}

	if r.PrivateMode {
		repo.Private = true
	} else {
		repo.Private = !repo_.Public
	}

	return repo, err
}

// Perm fetches the named repository from the remote system.
func (r *Gitlab) Perm(u *common.User, owner, name string) (*common.Perm, error) {
	key := fmt.Sprintf("%s/%s/%s", u.Login, owner, name)
	val, ok := r.cache.Get(key)
	if ok {
		return val.(*common.Perm), nil
	}

	client := NewClient(r.URL, u.Token, r.SkipVerify)
	id, err := GetProjectId(r, client, owner, name)
	if err != nil {
		return nil, err
	}

	repo, err := client.Project(id)
	if err != nil {
		return nil, err
	}
	m := &common.Perm{}
	m.Admin = IsAdmin(repo)
	m.Pull = IsRead(repo)
	m.Push = IsWrite(repo)
	r.cache.Add(key, m)
	return m, nil
}

// GetScript fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (r *Gitlab) Script(user *common.User, repo *common.Repo, build *common.Build) ([]byte, error) {
	var client = NewClient(r.URL, user.Token, r.SkipVerify)
	id, err := GetProjectId(r, client, repo.Owner, repo.Name)
	if err != nil {
		return nil, err
	}

	return client.RepoRawFile(id, build.Commit.Sha, ".drone.yml")
}

// NOTE Currently gitlab doesn't support status for commits and events,
//      also if we want get MR status in gitlab we need implement a special plugin for gitlab,
//      gitlab uses API to fetch build status on client side. But for now we skip this.
func (r *Gitlab) Status(u *common.User, repo *common.Repo, b *common.Build) error {
	return nil
}

// Netrc returns a .netrc file that can be used to clone
// private repositories from a remote system.
func (r *Gitlab) Netrc(u *common.User) (*common.Netrc, error) {
	url_, err := url.Parse(r.URL)
	if err != nil {
		return nil, err
	}
	netrc := &common.Netrc{}
	netrc.Login = "oauth2"
	netrc.Password = u.Token
	netrc.Machine = url_.Host
	return netrc, nil
}

// Activate activates a repository by adding a Post-commit hook and
// a Public Deploy key, if applicable.
func (r *Gitlab) Activate(user *common.User, repo *common.Repo, k *common.Keypair, link string) error {
	var client = NewClient(r.URL, user.Token, r.SkipVerify)
	id, err := GetProjectId(r, client, repo.Owner, repo.Name)
	if err != nil {
		return err
	}

	uri, err := url.Parse(link)
	if err != nil {
		return err
	}

	droneUrl := fmt.Sprintf("%s://%s", uri.Scheme, uri.Host)
	droneToken := uri.Query().Get("access_token")

	return client.AddDroneService(id, map[string]string{"token": droneToken, "drone_url": droneUrl})
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (r *Gitlab) Deactivate(user *common.User, repo *common.Repo, link string) error {
	var client = NewClient(r.URL, user.Token, r.SkipVerify)
	id, err := GetProjectId(r, client, repo.Owner, repo.Name)
	if err != nil {
		return err
	}

	return client.DeleteDroneService(id)
}

// ParseHook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (r *Gitlab) Hook(req *http.Request) (*common.Hook, error) {
	defer req.Body.Close()
	var payload, _ = ioutil.ReadAll(req.Body)
	var parsed, err = gogitlab.ParseHook(payload)
	if err != nil {
		return nil, err
	}

	switch parsed.ObjectKind {
	case "merge_request":
		if parsed.ObjectAttributes.State != "reopened" && parsed.ObjectAttributes.MergeStatus != "unchecked" ||
			parsed.ObjectAttributes.State != "opened" && parsed.ObjectAttributes.MergeStatus != "unchecked" {
			return nil, nil
		}

		return mergeRequest(parsed, req)
	case "push":
		if len(parsed.After) == 0 || parsed.TotalCommitsCount == 0 {
			return nil, nil
		}

		return push(parsed, req)
	default:
		return nil, nil
	}
}

func mergeRequest(parsed *gogitlab.HookPayload, req *http.Request) (*common.Hook, error) {
	var hook = new(common.Hook)

	re := regexp.MustCompile(".git$")

	hook.Repo = &common.Repo{}
	hook.Repo.Owner = req.FormValue("owner")
	hook.Repo.Name = req.FormValue("name")
	hook.Repo.FullName = fmt.Sprintf("%s/%s", hook.Repo.Owner, hook.Repo.Name)
	hook.Repo.Link = re.ReplaceAllString(parsed.ObjectAttributes.Target.HttpUrl, "$1")
	hook.Repo.Clone = parsed.ObjectAttributes.Target.HttpUrl
	hook.Repo.Branch = "master"

	hook.Commit = &common.Commit{}
	hook.Commit.Message = parsed.ObjectAttributes.LastCommit.Message
	hook.Commit.Sha = parsed.ObjectAttributes.LastCommit.Id
	hook.Commit.Remote = parsed.ObjectAttributes.Source.HttpUrl

	if parsed.ObjectAttributes.Source.HttpUrl == parsed.ObjectAttributes.Target.HttpUrl {
		hook.Commit.Ref = fmt.Sprintf("refs/heads/%s", parsed.ObjectAttributes.SourceBranch)
		hook.Commit.Branch = parsed.ObjectAttributes.SourceBranch
		hook.Commit.Timestamp = time.Now().UTC().Format("2006-01-02 15:04:05.000000000 +0000 MST")
	} else {
		hook.Commit.Ref = fmt.Sprintf("refs/merge-requests/%d/head", parsed.ObjectAttributes.IId)
		hook.Commit.Branch = parsed.ObjectAttributes.SourceBranch
		hook.Commit.Timestamp = time.Now().UTC().Format("2006-01-02 15:04:05.000000000 +0000 MST")
	}

	hook.Commit.Author = &common.Author{}
	hook.Commit.Author.Login = parsed.ObjectAttributes.LastCommit.Author.Name
	hook.Commit.Author.Email = parsed.ObjectAttributes.LastCommit.Author.Email

	hook.PullRequest = &common.PullRequest{}
	hook.PullRequest.Number = parsed.ObjectAttributes.IId
	hook.PullRequest.Title = parsed.ObjectAttributes.Description
	hook.PullRequest.Link = parsed.ObjectAttributes.URL

	return hook, nil
}

func push(parsed *gogitlab.HookPayload, req *http.Request) (*common.Hook, error) {
	var cloneUrl = parsed.Repository.GitHttpUrl

	var hook = new(common.Hook)
	hook.Repo = &common.Repo{}
	hook.Repo.Owner = req.FormValue("owner")
	hook.Repo.Name = req.FormValue("name")
	hook.Repo.Link = parsed.Repository.URL
	hook.Repo.Clone = cloneUrl
	hook.Repo.Branch = "master"

	switch parsed.Repository.VisibilityLevel {
	case 0:
		hook.Repo.Private = true
	case 10:
		hook.Repo.Private = true
	case 20:
		hook.Repo.Private = false
	}

	hook.Repo.FullName = fmt.Sprintf("%s/%s", req.FormValue("owner"), req.FormValue("name"))

	hook.Commit = &common.Commit{}
	hook.Commit.Sha = parsed.After
	hook.Commit.Branch = parsed.Branch()
	hook.Commit.Ref = parsed.Ref
	hook.Commit.Remote = cloneUrl

	var head = parsed.Head()
	hook.Commit.Message = head.Message
	hook.Commit.Timestamp = head.Timestamp
	hook.Commit.Author = &common.Author{}

	// extracts the commit author (ideally email)
	// from the post-commit hook
	switch {
	case head.Author != nil:
		hook.Commit.Author.Email = head.Author.Email
		hook.Commit.Author.Login = parsed.UserName
	case head.Author == nil:
		hook.Commit.Author.Login = parsed.UserName
	}

	return hook, nil
}

// ¯\_(ツ)_/¯
func (g *Gitlab) Oauth2Transport(r *http.Request) *oauth2.Transport {
	return &oauth2.Transport{
		Config: &oauth2.Config{
			ClientId:     g.Client,
			ClientSecret: g.Secret,
			Scope:        DefaultScope,
			AuthURL:      fmt.Sprintf("%s/oauth/authorize", g.URL),
			TokenURL:     fmt.Sprintf("%s/oauth/token", g.URL),
			RedirectURL:  fmt.Sprintf("%s/authorize", httputil.GetURL(r)),
			//settings.Server.Scheme, settings.Server.Hostname),
		},
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: g.SkipVerify},
		},
	}
}

// Accessor method, to allowed remote organizations field.
func (r *Gitlab) GetOrgs() []string {
	return r.AllowedOrgs
}

// Accessor method, to open field.
func (r *Gitlab) GetOpen() bool {
	return r.Open
}

// return default scope for GitHub
func (r *Gitlab) Scope() string {
	return DefaultScope
}
