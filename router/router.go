// Copyright 2018 Drone.IO Inc.
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

package router

import (
	"net/http"

	"github.com/dimfeld/httptreemux"
	"github.com/gin-gonic/gin"

	"github.com/drone/drone/router/middleware/header"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/router/middleware/token"
	"github.com/drone/drone/server"
	"github.com/drone/drone/server/debug"
	"github.com/drone/drone/server/metrics"
	"github.com/drone/drone/server/web"
)

// Load loads the router
func Load(mux *httptreemux.ContextMux, middleware ...gin.HandlerFunc) http.Handler {

	e := gin.New()
	e.Use(gin.Recovery())

	e.Use(header.NoCache)
	e.Use(header.Options)
	e.Use(header.Secure)
	e.Use(middleware...)
	e.Use(session.SetUser())
	e.Use(token.Refresh)

	e.NoRoute(func(c *gin.Context) {
		req := c.Request.WithContext(
			web.WithUser(
				c.Request.Context(),
				session.User(c),
			),
		)
		mux.ServeHTTP(c.Writer, req)
	})

	e.GET("/logout", server.GetLogout)
	e.GET("/login", server.HandleLogin)

	user := e.Group("/api/user")
	{
		user.Use(session.MustUser())
		user.GET("", server.GetSelf)
		user.GET("/feed", server.GetFeed)
		user.GET("/repos", server.GetRepos)
		user.POST("/token", server.PostToken)
		user.DELETE("/token", server.DeleteToken)
	}

	users := e.Group("/api/users")
	{
		users.Use(session.MustAdmin())
		users.GET("", server.GetUsers)
		users.POST("", server.PostUser)
		users.GET("/:login", server.GetUser)
		users.PATCH("/:login", server.PatchUser)
		users.DELETE("/:login", server.DeleteUser)
	}

	repo := e.Group("/api/repos/:owner/:name")
	{
		repo.Use(session.SetRepo())
		repo.Use(session.SetPerm())
		repo.Use(session.MustPull)

		repo.POST("", session.MustRepoAdmin(), server.PostRepo)
		repo.GET("", server.GetRepo)
		repo.GET("/builds", server.GetBuilds)
		repo.GET("/builds/:number", server.GetBuild)
		repo.GET("/logs/:number/:pid", server.GetProcLogs)
		repo.GET("/logs/:number/:pid/:proc", server.GetBuildLogs)

		repo.GET("/files/:number", server.FileList)
		repo.GET("/files/:number/:proc/*file", server.FileGet)

		// requires push permissions
		repo.GET("/secrets", session.MustPush, server.GetSecretList)
		repo.POST("/secrets", session.MustPush, server.PostSecret)
		repo.GET("/secrets/:secret", session.MustPush, server.GetSecret)
		repo.PATCH("/secrets/:secret", session.MustPush, server.PatchSecret)
		repo.DELETE("/secrets/:secret", session.MustPush, server.DeleteSecret)

		// requires push permissions
		repo.GET("/registry", session.MustPush, server.GetRegistryList)
		repo.POST("/registry", session.MustPush, server.PostRegistry)
		repo.GET("/registry/:registry", session.MustPush, server.GetRegistry)
		repo.PATCH("/registry/:registry", session.MustPush, server.PatchRegistry)
		repo.DELETE("/registry/:registry", session.MustPush, server.DeleteRegistry)

		// requires admin permissions
		repo.PATCH("", session.MustRepoAdmin(), server.PatchRepo)
		repo.DELETE("", session.MustRepoAdmin(), server.DeleteRepo)
		repo.POST("/chown", session.MustRepoAdmin(), server.ChownRepo)
		repo.POST("/repair", session.MustRepoAdmin(), server.RepairRepo)
		repo.POST("/move", session.MustRepoAdmin(), server.MoveRepo)

		repo.POST("/builds/:number", session.MustPush, server.PostBuild)
		repo.DELETE("/builds/:number", session.MustRepoAdmin(), server.ZombieKill)
		repo.POST("/builds/:number/approve", session.MustPush, server.PostApproval)
		repo.POST("/builds/:number/decline", session.MustPush, server.PostDecline)
		repo.DELETE("/builds/:number/:job", session.MustPush, server.DeleteBuild)
		repo.DELETE("/logs/:number", session.MustPush, server.DeleteBuildLogs)
	}

	badges := e.Group("/api/badges/:owner/:name")
	{
		badges.GET("/status.svg", server.GetBadge)
		badges.GET("/cc.xml", server.GetCC)
	}

	e.POST("/hook", server.PostHook)
	e.POST("/api/hook", server.PostHook)

	sse := e.Group("/stream")
	{
		sse.GET("/events", server.EventStreamSSE)
		sse.GET("/logs/:owner/:name/:build/:number",
			session.SetRepo(),
			session.SetPerm(),
			session.MustPull,
			server.LogStreamSSE,
		)
	}

	info := e.Group("/api/info")
	{
		info.GET("/queue",
			session.MustAdmin(),
			server.GetQueueInfo,
		)
	}

	auth := e.Group("/authorize")
	{
		auth.GET("", server.HandleAuth)
		auth.POST("", server.HandleAuth)
		auth.POST("/token", server.GetLoginToken)
	}

	builds := e.Group("/api/builds")
	{
		builds.Use(session.MustAdmin())
		builds.GET("/recent", server.GetRecentBuilds)
		builds.GET("", server.GetBuildQueue)
	}

	debugger := e.Group("/api/debug")
	{
		debugger.Use(session.MustAdmin())
		debugger.GET("/pprof/", debug.IndexHandler())
		debugger.GET("/pprof/heap", debug.HeapHandler())
		debugger.GET("/pprof/goroutine", debug.GoroutineHandler())
		debugger.GET("/pprof/block", debug.BlockHandler())
		debugger.GET("/pprof/threadcreate", debug.ThreadCreateHandler())
		debugger.GET("/pprof/cmdline", debug.CmdlineHandler())
		debugger.GET("/pprof/profile", debug.ProfileHandler())
		debugger.GET("/pprof/symbol", debug.SymbolHandler())
		debugger.POST("/pprof/symbol", debug.SymbolHandler())
		debugger.GET("/pprof/trace", debug.TraceHandler())
	}

	monitor := e.Group("/metrics")
	{
		monitor.GET("", metrics.PromHandler())
	}

	e.GET("/version", server.Version)
	e.GET("/healthz", server.Health)

	return e
}
