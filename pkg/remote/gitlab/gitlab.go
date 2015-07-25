package gitlab

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Bugagazavr/go-gitlab-client"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/hashicorp/golang-lru"
	"github.com/drone/drone/pkg/config"
	"github.com/drone/drone/pkg/oauth2"
	"github.com/drone/drone/pkg/remote"
	common "github.com/drone/drone/pkg/types"
	"github.com/drone/drone/pkg/utils/httputil"
)

const (
	DefaultScope = "repo"
)

const (
	StatusPending = "pending"
	StatusSuccess = "success"
	StatusFailure = "failure"
	StatusError   = "error"
)

const (
	DescPending = "this build is pending"
	DescSuccess = "the build was successful"
	DescFailure = "the build failed"
	DescError   = "oops, something went wrong"
)

type Gitlab struct {
	URL         string
	Client      string
	Secret      string
	AllowedOrgs []string
	Open        bool
	PrivateMode bool
	SkipVerify  bool

	cache *lru.Cache
}

func init() {
	remote.Register("gitlab", NewDriver)
}

func NewDriver(conf *config.Config) (remote.Remote, error) {
	var gitlab = Gitlab{
		URL:         conf.Gitlab.URL,
		Client:      conf.Gitlab.Client,
		Secret:      conf.Gitlab.Secret,
		AllowedOrgs: conf.Gitlab.Orgs,
		Open:        conf.Gitlab.Open,
		SkipVerify:  conf.Gitlab.SkipVerify,
	}
	var err error
	gitlab.cache, err = lru.New(1028)
	if err != nil {
		return nil, err
	}

	// the URL must NOT have a trailing slash
	if strings.HasSuffix(gitlab.URL, "/") {
		gitlab.URL = gitlab.URL[:len(gitlab.URL)-1]
	}
	return &gitlab, nil
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
	return &user, nil
}

// Orgs fetches the organizations for the given user.
func (r *Gitlab) Orgs(u *common.User) ([]string, error) {
	return nil, nil
}

// Repo fetches the named repository from the remote system.
func (r *Gitlab) Repo(u *common.User, owner, name string) (*common.Repo, error) {
	client := NewClient(r.URL, u.Token, r.SkipVerify)
	id := ns(owner, name)
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
	id := ns(owner, name)
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
	var path = ns(repo.Owner, repo.Name)
	return client.RepoRawFile(path, build.Commit.Sha, ".drone.yml")
}

func (r *Gitlab) Status(u *common.User, repo *common.Repo, b *common.Build) error {
	// how can I get Gitlab status
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
	netrc.Login = u.Token
	netrc.Password = "x-oauth-basic"
	netrc.Machine = url_.Host
	return netrc, nil
}

// Activate activates a repository by adding a Post-commit hook and
// a Public Deploy key, if applicable.
func (r *Gitlab) Activate(user *common.User, repo *common.Repo, keys *common.Keypair, link string) error {
	var client = NewClient(r.URL, user.Token, r.SkipVerify)
	var path = ns(repo.Owner, repo.Name)
	var title, err = GetKeyTitle(link)
	if err != nil {
		return err
	}

	// if the repository is private we'll need
	// to upload a github key to the repository
	if repo.Private {
		var err = client.AddProjectDeployKey(path, title, repo.Keys.Public)
		if err != nil {
			return err
		}
	}

	// append the repo owner / name to the hook url since gitlab
	// doesn't send this detail in the post-commit hook
	link += "?owner=" + repo.Owner + "&name=" + repo.Name

	// add the hook
	return client.AddProjectHook(path, link, true, false, true)
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (r *Gitlab) Deactivate(user *common.User, repo *common.Repo, link string) error {
	var client = NewClient(r.URL, user.Token, r.SkipVerify)
	var path = ns(repo.Owner, repo.Name)

	keys, err := client.ProjectDeployKeys(path)
	if err != nil {
		return err
	}
	var pubkey = strings.TrimSpace(repo.Keys.Public)
	for _, k := range keys {
		if pubkey == strings.TrimSpace(k.Key) {
			if err := client.RemoveProjectDeployKey(path, strconv.Itoa(k.Id)); err != nil {
				return err
			}
			break
		}
	}
	hooks, err := client.ProjectHooks(path)
	if err != nil {
		return err
	}
	link += "?owner=" + repo.Owner + "&name=" + repo.Name
	for _, h := range hooks {
		if link == h.Url {
			if err := client.RemoveProjectHook(path, strconv.Itoa(h.Id)); err != nil {
				return err
			}
			break
		}
	}
	return nil
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

	if len(parsed.After) == 0 || parsed.TotalCommitsCount == 0 {
		return nil, nil
	}

	if parsed.ObjectKind == "merge_request" {
		// TODO (bradrydzewski) figure out how to handle merge requests
		return nil, nil
	}

	if len(parsed.After) == 0 {
		return nil, nil
	}

	var hook = new(common.Hook)
	hook.Repo.Owner = req.FormValue("owner")
	hook.Repo.Name = req.FormValue("name")
	hook.Commit.Sha = parsed.After
	hook.Commit.Branch = parsed.Branch()

	var head = parsed.Head()
	hook.Commit.Message = head.Message
	hook.Commit.Timestamp = head.Timestamp

	// extracts the commit author (ideally email)
	// from the post-commit hook
	switch {
	case head.Author != nil:
		hook.Commit.Author.Login = head.Author.Email
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
			TokenURL:     fmt.Sprintf("%s/oauth/access_token", g.URL),
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

// getStatus is a helper functin that converts a Drone
// status to a GitHub status.
func getStatus(status string) string {
	switch status {
	case common.StatePending, common.StateRunning:
		return StatusPending
	case common.StateSuccess:
		return StatusSuccess
	case common.StateFailure:
		return StatusFailure
	case common.StateError, common.StateKilled:
		return StatusError
	default:
		return StatusError
	}
}

func New(conf *config.Config) *Gitlab {
	var gitlab = Gitlab{
		URL:         conf.Gitlab.URL,
		Client:      conf.Gitlab.Client,
		Secret:      conf.Gitlab.Secret,
		SkipVerify:  conf.Gitlab.SkipVerify,
		AllowedOrgs: conf.Gitlab.Orgs,
		Open:        conf.Gitlab.Open,
	}
	var err error
	gitlab.cache, err = lru.New(1028)
	if err != nil {
		panic(err)
	}

	// the URL must NOT have a trailing slash
	if strings.HasSuffix(gitlab.URL, "/") {
		gitlab.URL = gitlab.URL[:len(gitlab.URL)-1]
	}
	return &gitlab
}

// GetHost returns the hostname of this remote GitHub instance.
func (r *Gitlab) GetHost() string {
	uri, _ := url.Parse(r.URL)
	return uri.Host
}

// getDesc is a helper function that generates a description
// message for the build based on the status.
func getDesc(status string) string {
	switch status {
	case common.StatePending, common.StateRunning:
		return DescPending
	case common.StateSuccess:
		return DescSuccess
	case common.StateFailure:
		return DescFailure
	case common.StateError, common.StateKilled:
		return DescError
	default:
		return DescError
	}
}
