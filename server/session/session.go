package session

import (
	"net/http"
	"time"

	"code.google.com/p/go.net/context"
	"github.com/dgrijalva/jwt-go"
	"github.com/drone/drone/server/datastore"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
	"github.com/gorilla/securecookie"
)

// secret key used to create jwt
var secret = securecookie.GenerateRandomKey(32)

// GetUser gets the currently authenticated user for the
// http.Request. The user details will be stored as either
// a simple API token or JWT bearer token.
func GetUser(c context.Context, r *http.Request) *model.User {
	var token = r.FormValue("access_token")
	switch {
	case len(token) == 0:
		return nil
	case len(token) == 32:
		return getUserToken(c, r)
	default:
		return getUserBearer(c, r)
	}
}

// GenerateToken generates a JWT token for the user session
// that can be appended to the #access_token segment to
// facilitate client-based OAuth2.
func GenerateToken(c context.Context, r *http.Request, user *model.User) (string, error) {
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	token.Claims["user_id"] = user.ID
	token.Claims["audience"] = httputil.GetURL(r)
	token.Claims["expires"] = time.Now().UTC().Add(time.Hour * 72).Unix()
	return token.SignedString(secret)
}

// getUserToken gets the currently authenticated user for the given
// auth token.
func getUserToken(c context.Context, r *http.Request) *model.User {
	var token = r.FormValue("access_token")
	var user, _ = datastore.GetUserToken(c, token)
	return user
}

// getUserBearer gets the currently authenticated user for the given
// bearer token (JWT)
func getUserBearer(c context.Context, r *http.Request) *model.User {
	var tokenstr = r.FormValue("access_token")
	var token, err = jwt.Parse(tokenstr, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil || token.Valid {
		return nil
	}
	var userid, ok = token.Claims["user_id"].(int64)
	if !ok {
		return nil
	}
	var user, _ = datastore.GetUser(c, userid)
	return user
}
