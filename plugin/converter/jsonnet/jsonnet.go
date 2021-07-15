package jsonnet

import (
	"bytes"
	"context"
	"fmt"
	"github.com/drone/drone/handler/api/errors"
	"github.com/drone/go-scm/scm"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/drone/drone/core"

	"github.com/google/go-jsonnet"
)

const repo = "repo."
const build = "build."
const param = "param."

type importer struct {
	repo  *core.Repository
	build *core.Build

	// jsonnet does not cache file imports and may request
	// the same file multiple times. We cache the files to
	// duplicate API calls.
	cache map[string]jsonnet.Contents

	// limit the number of outbound requests. github limits
	// the number of api requests per hour, so we should
	// make sure that a single build does not abuse the api
	// by imporing dozens of files.
	limit int

	// counts the number of outbound requests. if the count
	// exceeds the limit, the importer will return errors.
	count int

	client *scm.Client
	user   *core.User
}

func (i *importer) Import(importedFrom, importedPath string) (contents jsonnet.Contents, foundAt string, err error) {
	if i.cache == nil {
		i.cache = map[string]jsonnet.Contents{}
	}

	// the import is relative to the imported from path. the
	// imported path must resolve to a filepath relative to
	// the root of the repository.
	importedPath = path.Join(
		path.Dir(importedFrom),
		importedPath,
	)

	if strings.HasPrefix(importedFrom, "../") {
		err = fmt.Errorf("jsonnet: cannot resolve import: %s", importedPath)
		return contents, foundAt, err
	}

	// if the contents exist in the cache, return the
	// cached item.
	if contents, ok := i.cache[importedPath]; ok {
		return contents, importedPath, nil
	}

	defer func() {
		i.count++
	}()

	// if the import limit is exceeded log an error message.
	if i.limit > 0 && i.count > i.limit {
		return contents, foundAt, errors.New("jsonnet: import limit exceeded")
	}

	ctx := context.WithValue(context.TODO(), scm.TokenKey{}, &scm.Token{
		Token:   i.user.Token,
		Refresh: i.user.Token,
		Expires: time.Time{},
	})

	find, _, err := i.client.Contents.Find(ctx, i.repo.Namespace+"/"+i.repo.Name, importedPath, i.build.Ref)

	if err != nil {
		return contents, foundAt, err
	}

	i.cache[importedPath] = jsonnet.MakeContents(string(find.Data))

	return i.cache[importedPath], importedPath, err
}

func Parse(req *core.ConvertArgs, client *scm.Client, template *core.Template, templateData map[string]interface{}) (string, error) {
	vm := jsonnet.MakeVM()
	vm.MaxStack = 500
	vm.StringOutput = false
	vm.ErrorFormatter.SetMaxStackTraceSize(20)
	vm.Importer(
		&importer{
			repo:   req.Repo,
			build:  req.Build,
			limit:  10,
			user:   req.User,
			client: client,
		},
	)

	//map build/repo parameters
	if req.Build != nil {
		mapBuild(req.Build, vm)
	}
	if req.Repo != nil {
		mapRepo(req.Repo, vm)
	}

	var jsonnetFile string
	var jsonnetFileName string
	if template != nil {
		jsonnetFile = template.Data
		jsonnetFileName = template.Name
	} else {
		jsonnetFile = req.Config.Data
		jsonnetFileName = req.Repo.Config
	}
	// map external inputs
	if len(templateData) != 0 {
		for k, v := range templateData {
			key := fmt.Sprintf("input." + k)
			val := fmt.Sprint(v)
			vm.ExtVar(key, val)
		}
	}

	// convert the jsonnet file to yaml
	buf := new(bytes.Buffer)
	docs, err := vm.EvaluateAnonymousSnippetStream(jsonnetFileName, jsonnetFile)
	if err != nil {
		doc, err2 := vm.EvaluateAnonymousSnippet(jsonnetFileName, jsonnetFile)
		if err2 != nil {
			return "", err
		}
		docs = append(docs, doc)
	}

	// the jsonnet vm returns a stream of yaml documents
	// that need to be combined into a single yaml file.
	for _, doc := range docs {
		buf.WriteString("---")
		buf.WriteString("\n")
		buf.WriteString(doc)
	}

	return buf.String(), nil
}

func mapBuild(v *core.Build, vm *jsonnet.VM) {
	vm.ExtVar(build+"event", v.Event)
	vm.ExtVar(build+"action", v.Action)
	vm.ExtVar(build+"environment", v.Deploy)
	vm.ExtVar(build+"link", v.Link)
	vm.ExtVar(build+"branch", v.Target)
	vm.ExtVar(build+"source", v.Source)
	vm.ExtVar(build+"before", v.Before)
	vm.ExtVar(build+"after", v.After)
	vm.ExtVar(build+"target", v.Target)
	vm.ExtVar(build+"ref", v.Ref)
	vm.ExtVar(build+"commit", v.After)
	vm.ExtVar(build+"ref", v.Ref)
	vm.ExtVar(build+"title", v.Title)
	vm.ExtVar(build+"message", v.Message)
	vm.ExtVar(build+"source_repo", v.Fork)
	vm.ExtVar(build+"author_login", v.Author)
	vm.ExtVar(build+"author_name", v.AuthorName)
	vm.ExtVar(build+"author_email", v.AuthorEmail)
	vm.ExtVar(build+"author_avatar", v.AuthorAvatar)
	vm.ExtVar(build+"sender", v.Sender)
	fromMap(v.Params, vm)
}

func mapRepo(v *core.Repository, vm *jsonnet.VM) {
	vm.ExtVar(repo+"uid", v.UID)
	vm.ExtVar(repo+"name", v.Name)
	vm.ExtVar(repo+"namespace", v.Namespace)
	vm.ExtVar(repo+"slug", v.Slug)
	vm.ExtVar(repo+"git_http_url", v.HTTPURL)
	vm.ExtVar(repo+"git_ssh_url", v.SSHURL)
	vm.ExtVar(repo+"link", v.Link)
	vm.ExtVar(repo+"branch", v.Branch)
	vm.ExtVar(repo+"config", v.Config)
	vm.ExtVar(repo+"private", strconv.FormatBool(v.Private))
	vm.ExtVar(repo+"visibility", v.Visibility)
	vm.ExtVar(repo+"active", strconv.FormatBool(v.Active))
	vm.ExtVar(repo+"trusted", strconv.FormatBool(v.Trusted))
	vm.ExtVar(repo+"protected", strconv.FormatBool(v.Protected))
	vm.ExtVar(repo+"ignore_forks", strconv.FormatBool(v.IgnoreForks))
	vm.ExtVar(repo+"ignore_pull_requests", strconv.FormatBool(v.IgnorePulls))
}

func fromMap(m map[string]string, vm *jsonnet.VM) {
	for k, v := range m {
		vm.ExtVar(build+param+k, v)
	}
}
