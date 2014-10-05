package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/drone/config"
	"github.com/drone/drone/server/handler"
	"github.com/drone/drone/server/middleware"
	"github.com/drone/drone/server/pubsub"
	"github.com/drone/drone/shared/build/log"

	"github.com/GeertJohan/go.rice"

	"code.google.com/p/go.net/context"
	webcontext "github.com/goji/context"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	_ "github.com/drone/drone/plugin/notify/email"
	"github.com/drone/drone/plugin/remote/bitbucket"
	"github.com/drone/drone/plugin/remote/github"
	"github.com/drone/drone/plugin/remote/gitlab"
	"github.com/drone/drone/server/blobstore"
	"github.com/drone/drone/server/capability"
	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/server/datastore/database"
	"github.com/drone/drone/server/worker/director"
	"github.com/drone/drone/server/worker/docker"
	"github.com/drone/drone/server/worker/pool"
)

var (
	// port the server will run on
	port string

	// database driver used to connect to the database
	driver string

	// driver specific connection information. In this
	// case, it should be the location of the SQLite file
	datasource string

	// optional flags for tls listener
	sslcert string
	sslkey  string

	// commit sha for the current build.
	version  string = "0.3-dev"
	revision string

	conf   string
	prefix string

	open bool

	// worker pool
	workers *pool.Pool

	// director
	worker *director.Director

	pub *pubsub.PubSub

	nodes StringArr

	db *sql.DB

	caps map[string]bool
)

func main() {
	log.SetPriority(log.LOG_NOTICE)

	flag.StringVar(&conf, "config", "", "")
	flag.StringVar(&prefix, "prefix", "DRONE_", "")
	flag.Parse()

	config.StringVar(&datasource, "database-source", "drone.sqlite")
	config.StringVar(&driver, "database-driver", "sqlite3")
	config.Var(&nodes, "worker-nodes")
	config.BoolVar(&open, "registration-open", false)
	config.SetPrefix(prefix)
	if err := config.Parse(conf); err != nil {
		fmt.Println("Error parsing config", err)
	}

	// setup the remote services
	bitbucket.Register()
	github.Register()
	gitlab.Register()

	caps = map[string]bool{}
	caps[capability.Registration] = open

	// setup the database and cancel all pending
	// commits in the system.
	db = database.MustConnect(driver, datasource)
	go database.NewCommitstore(db).KillCommits()

	// Create the worker, director and builders
	workers = pool.New()
	worker = director.New()

	if nodes == nil || len(nodes) == 0 {
		workers.Allocate(docker.New())
		workers.Allocate(docker.New())
	} else {
		for _, node := range nodes {
			workers.Allocate(docker.NewHost(node))
		}
	}

	pub = pubsub.NewPubSub()

	goji.Get("/api/logins", handler.GetLoginList)
	goji.Get("/api/stream/stdout/:id", handler.WsConsole)
	goji.Get("/api/stream/user", handler.WsUser)
	goji.Get("/api/auth/:host", handler.GetLogin)
	goji.Post("/api/auth/:host", handler.GetLogin)
	goji.Get("/api/badge/:host/:owner/:name/status.svg", handler.GetBadge)
	goji.Get("/api/badge/:host/:owner/:name/cc.xml", handler.GetCC)
	goji.Get("/api/hook/:host", handler.PostHook)
	goji.Put("/api/hook/:host", handler.PostHook)
	goji.Post("/api/hook/:host", handler.PostHook)

	repos := web.New()
	repos.Use(middleware.SetRepo)
	repos.Use(middleware.RequireRepoRead)
	repos.Use(middleware.RequireRepoAdmin)
	repos.Get("/api/repos/:host/:owner/:name/branches/:branch/commits/:commit/console", handler.GetOutput)
	repos.Get("/api/repos/:host/:owner/:name/branches/:branch/commits/:commit", handler.GetCommit)
	repos.Post("/api/repos/:host/:owner/:name/branches/:branch/commits/:commit", handler.PostCommit)
	repos.Get("/api/repos/:host/:owner/:name/commits", handler.GetCommitList)
	repos.Get("/api/repos/:host/:owner/:name", handler.GetRepo)
	repos.Put("/api/repos/:host/:owner/:name", handler.PutRepo)
	repos.Post("/api/repos/:host/:owner/:name", handler.PostRepo)
	repos.Delete("/api/repos/:host/:owner/:name", handler.DelRepo)
	goji.Handle("/api/repos/:host/:owner/:name*", repos)

	users := web.New()
	users.Use(middleware.RequireUserAdmin)
	users.Get("/api/users/:host/:login", handler.GetUser)
	users.Post("/api/users/:host/:login", handler.PostUser)
	users.Delete("/api/users/:host/:login", handler.DelUser)
	users.Get("/api/users", handler.GetUserList)
	goji.Handle("/api/users*", users)

	user := web.New()
	user.Use(middleware.RequireUser)
	user.Get("/api/user/feed", handler.GetUserFeed)
	user.Get("/api/user/repos", handler.GetUserRepos)
	user.Get("/api/user", handler.GetUserCurrent)
	user.Put("/api/user", handler.PutUser)
	goji.Handle("/api/user*", user)

	work := web.New()
	work.Use(middleware.RequireUserAdmin)
	work.Get("/api/work/started", handler.GetWorkStarted)
	work.Get("/api/work/pending", handler.GetWorkPending)
	work.Get("/api/work/assignments", handler.GetWorkAssigned)
	work.Get("/api/workers", handler.GetWorkers)
	goji.Handle("/api/work*", work)

	// Include static resources
	assets := rice.MustFindBox("app").HTTPBox()
	assetserve := http.FileServer(rice.MustFindBox("app").HTTPBox())
	http.Handle("/static/", http.StripPrefix("/static", assetserve))
	goji.Get("/*", func(c web.C, w http.ResponseWriter, r *http.Request) {
		w.Write(assets.MustBytes("index.html"))
	})

	// Add middleware and serve
	goji.Use(ContextMiddleware)
	goji.Use(middleware.SetHeaders)
	goji.Use(middleware.SetUser)
	goji.Serve()
}

// ContextMiddleware creates a new go.net/context and
// injects into the current goji context.
func ContextMiddleware(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var ctx = context.Background()
		ctx = datastore.NewContext(ctx, database.NewDatastore(db))
		ctx = blobstore.NewContext(ctx, database.NewBlobstore(db))
		ctx = pool.NewContext(ctx, workers)
		ctx = director.NewContext(ctx, worker)
		ctx = pubsub.NewContext(ctx, pub)
		ctx = capability.NewContext(ctx, caps)

		// add the context to the goji web context
		webcontext.Set(c, ctx)
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

type StringArr []string

func (s *StringArr) String() string {
	return fmt.Sprint(*s)
}

func (s *StringArr) Set(value string) error {
	for _, str := range strings.Split(value, ",") {
		str = strings.TrimSpace(str)
		*s = append(*s, str)
	}
	return nil
}
