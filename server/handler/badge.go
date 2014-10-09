package handler

import (
	"encoding/xml"
	"net/http"

	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
	"github.com/goji/context"
	"github.com/zenazn/goji/web"
)

// badges that indicate the current build status for a repository
// and branch combination.
var (
	badgeSuccess = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="91" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="91" height="18" fill="#555"/><rect rx="4" x="37" width="54" height="18" fill="#4c1"/><path fill="#4c1" d="M37 0h4v18h-4z"/><rect rx="4" width="91" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="63" y="13" fill="#010101" fill-opacity=".3">success</text><text x="63" y="12">success</text></g></svg>`)
	badgeFailure = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="83" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="83" height="18" fill="#555"/><rect rx="4" x="37" width="46" height="18" fill="#e05d44"/><path fill="#e05d44" d="M37 0h4v18h-4z"/><rect rx="4" width="83" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="59" y="13" fill="#010101" fill-opacity=".3">failure</text><text x="59" y="12">failure</text></g></svg>`)
	badgeStarted = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="87" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="87" height="18" fill="#555"/><rect rx="4" x="37" width="50" height="18" fill="#dfb317"/><path fill="#dfb317" d="M37 0h4v18h-4z"/><rect rx="4" width="87" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="61" y="13" fill="#010101" fill-opacity=".3">started</text><text x="61" y="12">started</text></g></svg>`)
	badgeError   = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="76" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="76" height="18" fill="#555"/><rect rx="4" x="37" width="39" height="18" fill="#9f9f9f"/><path fill="#9f9f9f" d="M37 0h4v18h-4z"/><rect rx="4" width="76" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="55.5" y="13" fill="#010101" fill-opacity=".3">error</text><text x="55.5" y="12">error</text></g></svg>`)
	badgeNone    = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="75" height="18"><linearGradient id="a" x2="0" y2="100%"><stop offset="0" stop-color="#fff" stop-opacity=".7"/><stop offset=".1" stop-color="#aaa" stop-opacity=".1"/><stop offset=".9" stop-opacity=".3"/><stop offset="1" stop-opacity=".5"/></linearGradient><rect rx="4" width="75" height="18" fill="#555"/><rect rx="4" x="37" width="38" height="18" fill="#9f9f9f"/><path fill="#9f9f9f" d="M37 0h4v18h-4z"/><rect rx="4" width="75" height="18" fill="url(#a)"/><g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11"><text x="19.5" y="13" fill="#010101" fill-opacity=".3">build</text><text x="19.5" y="12">build</text><text x="55" y="13" fill="#010101" fill-opacity=".3">none</text><text x="55" y="12">none</text></g></svg>`)
)

// GetBadge accepts a request to retrieve the named
// repo and branhes latest build details from the datastore
// and return an SVG badges representing the build results.
//
//     GET /api/badge/:host/:owner/:name/status.svg
//
func GetBadge(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var (
		host   = c.URLParams["host"]
		owner  = c.URLParams["owner"]
		name   = c.URLParams["name"]
		branch = c.URLParams["branch"]
	)

	repo, err := datastore.GetRepoName(ctx, host, owner, name)
	if err != nil {
		w.Write(badgeNone)
		return
	}
	if len(branch) == 0 {
		branch = model.DefaultBranch
	}
	commit, _ := datastore.GetCommitLast(ctx, repo, branch)

	// if no commit was found then display
	// the 'none' badge, instead of throwing
	// an error response
	if commit == nil {
		w.Write(badgeNone)
		return
	}

	switch commit.Status {
	case model.StatusSuccess:
		w.Write(badgeSuccess)
	case model.StatusFailure:
		w.Write(badgeFailure)
	case model.StatusError:
		w.Write(badgeError)
	case model.StatusEnqueue, model.StatusStarted:
		w.Write(badgeStarted)
	default:
		w.Write(badgeNone)
	}
}

// GetCC accepts a request to retrieve the latest build
// status for the given repository from the datastore and
// in CCTray XML format.
//
//     GET /api/badge/:host/:owner/:name/cc.xml
//
func GetCC(c web.C, w http.ResponseWriter, r *http.Request) {
	var ctx = context.FromC(c)
	var (
		host  = c.URLParams["host"]
		owner = c.URLParams["owner"]
		name  = c.URLParams["name"]
	)

	w.Header().Set("Content-Type", "image/svg+xml")

	repo, err := datastore.GetRepoName(ctx, host, owner, name)
	if err != nil {
		w.Write(badgeNone)
		return
	}
	commits, err := datastore.GetCommitList(ctx, repo)
	if err != nil || len(commits) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var link = httputil.GetURL(r) + "/" + repo.Host + "/" + repo.Owner + "/" + repo.Name
	var cc = model.NewCC(repo, commits[0], link)
	xml.NewEncoder(w).Encode(cc)
}
