// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"net/http"
	"os"

	"github.com/drone/drone/core"
	"github.com/drone/drone/handler/api/acl"
	"github.com/drone/drone/handler/api/auth"
	"github.com/drone/drone/handler/api/badge"
	globalbuilds "github.com/drone/drone/handler/api/builds"
	"github.com/drone/drone/handler/api/ccmenu"
	"github.com/drone/drone/handler/api/events"
	"github.com/drone/drone/handler/api/queue"
	"github.com/drone/drone/handler/api/repos"
	"github.com/drone/drone/handler/api/repos/builds"
	"github.com/drone/drone/handler/api/repos/builds/branches"
	"github.com/drone/drone/handler/api/repos/builds/deploys"
	"github.com/drone/drone/handler/api/repos/builds/logs"
	"github.com/drone/drone/handler/api/repos/builds/pulls"
	"github.com/drone/drone/handler/api/repos/builds/stages"
	"github.com/drone/drone/handler/api/repos/collabs"
	"github.com/drone/drone/handler/api/repos/crons"
	"github.com/drone/drone/handler/api/repos/encrypt"
	"github.com/drone/drone/handler/api/repos/secrets"
	"github.com/drone/drone/handler/api/repos/sign"
	globalsecrets "github.com/drone/drone/handler/api/secrets"
	"github.com/drone/drone/handler/api/system"
	"github.com/drone/drone/handler/api/template"
	"github.com/drone/drone/handler/api/user"
	"github.com/drone/drone/handler/api/user/remote"
	"github.com/drone/drone/handler/api/users"
	"github.com/drone/drone/logger"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

var corsOpts = cors.Options{
	AllowedOrigins:   []string{"*"},
	AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
	AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
	ExposedHeaders:   []string{"Link"},
	AllowCredentials: true,
	MaxAge:           300,
}

func New(
	builds core.BuildStore,
	commits core.CommitService,
	cron core.CronStore,
	events core.Pubsub,
	globals core.GlobalSecretStore,
	hooks core.HookService,
	logs core.LogStore,
	license *core.License,
	licenses core.LicenseService,
	orgs core.OrganizationService,
	perms core.PermStore,
	repos core.RepositoryStore,
	repoz core.RepositoryService,
	scheduler core.Scheduler,
	secrets core.SecretStore,
	stages core.StageStore,
	steps core.StepStore,
	status core.StatusService,
	session core.Session,
	stream core.LogStream,
	syncer core.Syncer,
	system *core.System,
	template core.TemplateStore,
	transferer core.Transferer,
	triggerer core.Triggerer,
	users core.UserStore,
	userz core.UserService,
	webhook core.WebhookSender,
) Server {
	return Server{
		Builds:     builds,
		Cron:       cron,
		Commits:    commits,
		Events:     events,
		Globals:    globals,
		Hooks:      hooks,
		Logs:       logs,
		License:    license,
		Licenses:   licenses,
		Orgs:       orgs,
		Perms:      perms,
		Repos:      repos,
		Repoz:      repoz,
		Scheduler:  scheduler,
		Secrets:    secrets,
		Stages:     stages,
		Steps:      steps,
		Status:     status,
		Session:    session,
		Stream:     stream,
		Syncer:     syncer,
		System:     system,
		Template:   template,
		Transferer: transferer,
		Triggerer:  triggerer,
		Users:      users,
		Userz:      userz,
		Webhook:    webhook,
	}
}

// Server is a http.Handler which exposes drone functionality over HTTP.
type Server struct {
	Builds     core.BuildStore
	Cron       core.CronStore
	Commits    core.CommitService
	Events     core.Pubsub
	Globals    core.GlobalSecretStore
	Hooks      core.HookService
	Logs       core.LogStore
	License    *core.License
	Licenses   core.LicenseService
	Orgs       core.OrganizationService
	Perms      core.PermStore
	Repos      core.RepositoryStore
	Repoz      core.RepositoryService
	Scheduler  core.Scheduler
	Secrets    core.SecretStore
	Stages     core.StageStore
	Steps      core.StepStore
	Status     core.StatusService
	Session    core.Session
	Stream     core.LogStream
	Syncer     core.Syncer
	System     *core.System
	Template   core.TemplateStore
	Transferer core.Transferer
	Triggerer  core.Triggerer
	Users      core.UserStore
	Userz      core.UserService
	Webhook    core.WebhookSender
	Private    bool
}

// Handler returns an http.Handler
func (s Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.NoCache)
	r.Use(logger.Middleware)
	r.Use(auth.HandleAuthentication(s.Session))

	cors := cors.New(corsOpts)
	r.Use(cors.Handler)

	r.Route("/repos", func(r chi.Router) {
		// temporary workaround to enable private mode
		// for the drone server.
		if os.Getenv("DRONE_SERVER_PRIVATE_MODE") == "true" {
			r.Use(acl.AuthorizeUser)
		}

		r.With(
			acl.AuthorizeAdmin,
		).Get("/", repos.HandleAll(s.Repos))

		r.Route("/{owner}/{name}", func(r chi.Router) {
			r.Use(acl.InjectRepository(s.Repoz, s.Repos, s.Perms))
			r.Use(acl.CheckReadAccess())

			r.Get("/", repos.HandleFind())
			r.With(
				acl.CheckAdminAccess(),
			).Patch("/", repos.HandleUpdate(s.Repos))
			r.With(
				acl.CheckAdminAccess(),
			).Post("/", repos.HandleEnable(s.Hooks, s.Repos, s.Webhook))
			r.With(
				acl.CheckAdminAccess(),
			).Delete("/", repos.HandleDisable(s.Users, s.Hooks, s.Repos, s.Webhook))
			r.With(
				acl.CheckAdminAccess(),
			).Post("/chown", repos.HandleChown(s.Repos))
			r.With(
				acl.CheckAdminAccess(),
			).Post("/repair", repos.HandleRepair(s.Hooks, s.Repoz, s.Repos, s.Users, s.System.Link))

			r.Route("/builds", func(r chi.Router) {
				r.Get("/", builds.HandleList(s.Repos, s.Builds))
				r.With(acl.CheckWriteAccess()).Post("/", builds.HandleCreate(s.Users, s.Repos, s.Commits, s.Triggerer))

				r.Get("/branches", branches.HandleList(s.Repos, s.Builds))
				r.With(acl.CheckWriteAccess()).Delete("/branches/*", branches.HandleDelete(s.Repos, s.Builds))

				r.Get("/pulls", pulls.HandleList(s.Repos, s.Builds))
				r.With(acl.CheckWriteAccess()).Delete("/pulls/{pull}", pulls.HandleDelete(s.Repos, s.Builds))

				r.Get("/deployments", deploys.HandleList(s.Repos, s.Builds))
				r.With(acl.CheckWriteAccess()).Delete("/deployments/*", deploys.HandleDelete(s.Repos, s.Builds))

				r.Get("/latest", builds.HandleLast(s.Repos, s.Builds, s.Stages))
				r.Get("/{number}", builds.HandleFind(s.Repos, s.Builds, s.Stages))
				r.Get("/{number}/logs/{stage}/{step}", logs.HandleFind(s.Repos, s.Builds, s.Stages, s.Steps, s.Logs))

				r.With(
					acl.CheckWriteAccess(),
				).Post("/{number}", builds.HandleRetry(s.Repos, s.Builds, s.Triggerer))

				r.With(
					acl.CheckWriteAccess(),
				).Delete("/{number}", builds.HandleCancel(s.Users, s.Repos, s.Builds, s.Stages, s.Steps, s.Status, s.Scheduler, s.Webhook))

				r.With(
					acl.CheckWriteAccess(),
				).Post("/{number}/promote", builds.HandlePromote(s.Repos, s.Builds, s.Triggerer))

				r.With(
					acl.CheckWriteAccess(),
				).Post("/{number}/rollback", builds.HandleRollback(s.Repos, s.Builds, s.Triggerer))

				r.With(
					acl.CheckAdminAccess(),
				).Post("/{number}/decline/{stage}", stages.HandleDecline(s.Repos, s.Builds, s.Stages))

				r.With(
					acl.CheckAdminAccess(),
				).Post("/{number}/approve/{stage}", stages.HandleApprove(s.Repos, s.Builds, s.Stages, s.Scheduler))

				r.With(
					acl.CheckAdminAccess(),
				).Delete("/{number}/logs/{stage}/{step}", logs.HandleDelete(s.Repos, s.Builds, s.Stages, s.Steps, s.Logs))

				r.With(
					acl.CheckAdminAccess(),
				).Delete("/", builds.HandlePurge(s.Repos, s.Builds))
			})

			r.Route("/secrets", func(r chi.Router) {
				r.Use(acl.CheckWriteAccess())
				r.Get("/", secrets.HandleList(s.Repos, s.Secrets))
				r.Post("/", secrets.HandleCreate(s.Repos, s.Secrets))
				r.Get("/{secret}", secrets.HandleFind(s.Repos, s.Secrets))
				r.Patch("/{secret}", secrets.HandleUpdate(s.Repos, s.Secrets))
				r.Delete("/{secret}", secrets.HandleDelete(s.Repos, s.Secrets))
			})

			r.Route("/sign", func(r chi.Router) {
				r.Use(acl.CheckWriteAccess())
				r.Post("/", sign.HandleSign(s.Repos))
			})

			r.Route("/encrypt", func(r chi.Router) {
				r.Use(acl.CheckWriteAccess())
				r.Post("/", encrypt.Handler(s.Repos))
				r.Post("/secret", encrypt.Handler(s.Repos))
			})

			r.Route("/cron", func(r chi.Router) {
				r.Use(acl.CheckWriteAccess())
				r.Post("/", crons.HandleCreate(s.Repos, s.Cron))
				r.Get("/", crons.HandleList(s.Repos, s.Cron))
				r.Get("/{cron}", crons.HandleFind(s.Repos, s.Cron))
				r.Post("/{cron}", crons.HandleExec(s.Users, s.Repos, s.Cron, s.Commits, s.Triggerer))
				r.Patch("/{cron}", crons.HandleUpdate(s.Repos, s.Cron))
				r.Delete("/{cron}", crons.HandleDelete(s.Repos, s.Cron))
			})

			r.Route("/collaborators", func(r chi.Router) {
				r.Get("/", collabs.HandleList(s.Repos, s.Perms))
				r.Get("/{member}", collabs.HandleFind(s.Users, s.Repos, s.Perms))
				r.With(
					acl.CheckAdminAccess(),
				).Delete("/{member}", collabs.HandleDelete(s.Users, s.Repos, s.Perms))
			})
		})
	})

	r.Route("/badges/{owner}/{name}", func(r chi.Router) {
		r.Get("/status.svg", badge.Handler(s.Repos, s.Builds))
		r.With(
			acl.InjectRepository(s.Repoz, s.Repos, s.Perms),
			acl.CheckReadAccess(),
		).Get("/cc.xml", ccmenu.Handler(s.Repos, s.Builds, s.System.Link))
	})

	r.Route("/queue", func(r chi.Router) {
		r.Use(acl.AuthorizeAdmin)
		r.Get("/", queue.HandleItems(s.Stages))
		r.Post("/", queue.HandleResume(s.Scheduler))
		r.Delete("/", queue.HandlePause(s.Scheduler))
	})

	r.Route("/user", func(r chi.Router) {
		r.Use(acl.AuthorizeUser)
		r.Get("/", user.HandleFind())
		r.Patch("/", user.HandleUpdate(s.Users))
		r.Post("/token", user.HandleToken(s.Users))
		r.Get("/repos", user.HandleRepos(s.Repos))
		r.Post("/repos", user.HandleSync(s.Syncer, s.Repos))

		// TODO(bradrydzewski) finalize the name for this endpoint.
		r.Get("/builds", user.HandleRecent(s.Repos))
		r.Get("/builds/recent", user.HandleRecent(s.Repos))

		// expose remote endpoints (e.g. to github)
		r.Get("/remote/repos", remote.HandleRepos(s.Repoz))
		r.Get("/remote/repos/{owner}/{name}", remote.HandleRepo(s.Repoz))
	})

	r.Route("/users", func(r chi.Router) {
		r.Use(acl.AuthorizeAdmin)
		r.Get("/", users.HandleList(s.Users))
		r.Post("/", users.HandleCreate(s.Users, s.Userz, s.Webhook))
		r.Get("/{user}", users.HandleFind(s.Users))
		r.Patch("/{user}", users.HandleUpdate(s.Users, s.Transferer))
		r.Delete("/{user}", users.HandleDelete(s.Users, s.Transferer, s.Webhook))
		r.Get("/{user}/repos", users.HandleRepoList(s.Users, s.Repos))
	})

	r.Route("/stream", func(r chi.Router) {
		r.Get("/", events.HandleGlobal(s.Repos, s.Events))

		r.Route("/{owner}/{name}", func(r chi.Router) {
			r.Use(acl.InjectRepository(s.Repoz, s.Repos, s.Perms))
			r.Use(acl.CheckReadAccess())

			r.Get("/", events.HandleEvents(s.Repos, s.Events))
			r.Get("/{number}/{stage}/{step}", events.HandleLogStream(s.Repos, s.Builds, s.Stages, s.Steps, s.Stream))
		})
	})

	r.Route("/builds", func(r chi.Router) {
		r.Use(acl.AuthorizeAdmin)
		r.Get("/incomplete", globalbuilds.HandleIncomplete(s.Repos))
	})

	r.Route("/secrets", func(r chi.Router) {
		r.With(acl.AuthorizeAdmin).Get("/", globalsecrets.HandleAll(s.Globals))
		r.With(acl.CheckMembership(s.Orgs, false)).Get("/{namespace}", globalsecrets.HandleList(s.Globals))
		r.With(acl.CheckMembership(s.Orgs, true)).Post("/{namespace}", globalsecrets.HandleCreate(s.Globals))
		r.With(acl.CheckMembership(s.Orgs, false)).Get("/{namespace}/{name}", globalsecrets.HandleFind(s.Globals))
		r.With(acl.CheckMembership(s.Orgs, true)).Post("/{namespace}/{name}", globalsecrets.HandleUpdate(s.Globals))
		r.With(acl.CheckMembership(s.Orgs, true)).Patch("/{namespace}/{name}", globalsecrets.HandleUpdate(s.Globals))
		r.With(acl.CheckMembership(s.Orgs, true)).Delete("/{namespace}/{name}", globalsecrets.HandleDelete(s.Globals))
	})

	r.Route("/templates", func(r chi.Router) {
		r.With(acl.CheckMembership(s.Orgs, false)).Get("/", template.HandleListAll(s.Template))
		r.With(acl.CheckMembership(s.Orgs, true)).Post("/", template.HandleCreate(s.Template))
		r.With(acl.CheckMembership(s.Orgs, false)).Get("/{namespace}", template.HandleList(s.Template))
		r.With(acl.CheckMembership(s.Orgs, false)).Get("/{namespace}/{name}", template.HandleFind(s.Template))
		r.With(acl.CheckMembership(s.Orgs, true)).Put("/{namespace}/{name}", template.HandleUpdate(s.Template))
		r.With(acl.CheckMembership(s.Orgs, true)).Patch("/{namespace}/{name}", template.HandleUpdate(s.Template))
		r.With(acl.CheckMembership(s.Orgs, true)).Delete("/{namespace}/{name}", template.HandleDelete(s.Template))
	})

	r.Route("/system", func(r chi.Router) {
		r.Use(acl.AuthorizeAdmin)
		// r.Get("/license", system.HandleLicense())
		// r.Get("/limits", system.HandleLimits())
		r.Get("/stats", system.HandleStats(
			s.Builds,
			s.Stages,
			s.Users,
			s.Repos,
			s.Events,
			s.Stream,
		))
	})

	return r
}
