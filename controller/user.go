package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/context"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/shared/token"
)

func GetSelf(c *gin.Context) {
	c.IndentedJSON(200, session.User(c))
}

func GetFeed(c *gin.Context) {
	user := session.User(c)
	db := context.Database(c)
	feed, err := model.GetUserFeed(db, user, 25, 0)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.IndentedJSON(http.StatusOK, feed)
}

func GetRepos(c *gin.Context) {
	user := session.User(c)
	db := context.Database(c)
	repos, err := model.GetRepoList(db, user)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.IndentedJSON(http.StatusOK, repos)
}

func GetRemoteRepos(c *gin.Context) {
	user := session.User(c)
	remote := context.Remote(c)

	reposv, ok := c.Get("repos")
	if ok {
		c.IndentedJSON(http.StatusOK, reposv)
		return
	}

	repos, err := remote.Repos(user)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Set("repos", repos)
	c.IndentedJSON(http.StatusOK, repos)
}

func PostToken(c *gin.Context) {
	user := session.User(c)

	token := token.New(token.UserToken, user.Login)
	tokenstr, err := token.Sign(user.Hash)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	} else {
		c.String(http.StatusOK, tokenstr)
	}
}
