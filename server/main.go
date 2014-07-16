package main

import (
	"database/sql"
	"flag"
	"net/http"
	"runtime"
	"strings"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/database/schema"
	"github.com/drone/drone/server/handler"
	"github.com/drone/drone/server/pubsub"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/server/worker"
	"github.com/drone/drone/shared/build/log"
	"github.com/drone/drone/shared/model"

	"github.com/gorilla/pat"
	//"github.com/justinas/nosurf"
	"github.com/GeertJohan/go.rice"
	_ "github.com/mattn/go-sqlite3"
	"github.com/russross/meddler"

	_ "github.com/drone/drone/plugin/remote/bitbucket"
	_ "github.com/drone/drone/plugin/remote/github"
	_ "github.com/drone/drone/plugin/remote/gitlab"
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

	// Number of concurrent build workers to run
	// default to number of CPUs on machine
	workers int
)

func main() {

	log.SetPriority(log.LOG_NOTICE)

	flag.StringVar(&port, "port", ":8080", "")
	flag.StringVar(&driver, "driver", "sqlite3", "")
	flag.StringVar(&datasource, "datasource", "drone.sqlite", "")
	flag.StringVar(&sslcert, "sslcert", "", "")
	flag.StringVar(&sslkey, "sslkey", "", "")
	flag.IntVar(&workers, "workers", runtime.NumCPU(), "")
	flag.Parse()

	// setup the database
	meddler.Default = meddler.SQLite
	db, _ := sql.Open(driver, datasource)
	schema.Load(db)

	// setup the database managers
	repos := database.NewRepoManager(db)
	users := database.NewUserManager(db)
	perms := database.NewPermManager(db)
	commits := database.NewCommitManager(db)
	servers := database.NewServerManager(db)
	remotes := database.NewRemoteManager(db)
	//configs := database.NewConfigManager(filepath.Join(home, "config.toml"))

	// message broker
	pubsub := pubsub.NewPubSub()

	// cancel all previously running builds
	go commits.CancelAll()

	queue := make(chan *model.Request)
	workerc := make(chan chan *model.Request)
	worker.NewDispatch(queue, workerc).Start()

	// there must be a minimum of 1 worker
	if workers <= 0 {
		workers = 1
	}

	// create the specified number of worker nodes
	for i := 0; i < workers; i++ {
		worker.NewWorker(workerc, users, repos, commits, pubsub, &model.Server{}).Start()
	}

	// setup the session managers
	sess := session.NewSession(users)

	// setup the router and register routes
	router := pat.New()
	handler.NewUsersHandler(users, sess).Register(router)
	handler.NewUserHandler(users, repos, commits, sess).Register(router)
	handler.NewHookHandler(users, repos, commits, remotes, queue).Register(router)
	handler.NewLoginHandler(users, repos, perms, sess, remotes).Register(router)
	handler.NewCommitHandler(repos, commits, perms, sess, queue).Register(router)
	handler.NewRepoHandler(repos, commits, perms, sess, remotes).Register(router)
	handler.NewBadgeHandler(repos, commits).Register(router)
	handler.NewServerHandler(servers, sess).Register(router)
	handler.NewRemoteHandler(users, remotes, sess).Register(router)
	handler.NewWsHandler(repos, commits, perms, sess, pubsub).Register(router)

	box := rice.MustFindBox("app/")
	fserver := http.FileServer(box.HTTPBox())
	index, _ := box.Bytes("index.html")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/favicon.ico"),
			strings.HasPrefix(r.URL.Path, "/scripts/"),
			strings.HasPrefix(r.URL.Path, "/styles/"),
			strings.HasPrefix(r.URL.Path, "/views/"):
			fserver.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, "/logout"),
			strings.HasPrefix(r.URL.Path, "/login/"),
			strings.HasPrefix(r.URL.Path, "/v1/"),
			strings.HasPrefix(r.URL.Path, "/ws/"):
			router.ServeHTTP(w, r)
		default:
			w.Write(index)
		}
	})

	// start webserver using HTTPS or HTTP
	if len(sslcert) != 0 {
		panic(http.ListenAndServeTLS(port, sslcert, sslkey, nil))
	} else {
		panic(http.ListenAndServe(port, nil))
	}
}

func init() {

}

func init_flags() {

}

func init_database() {

}

func init_workers() {

}

func init_handlers() {

}
