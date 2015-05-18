package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/drone/drone/pkg/server/recorder"
	"github.com/drone/drone/pkg/server/session"
	"github.com/drone/drone/pkg/settings"
	"github.com/drone/drone/pkg/store/mock"
	"github.com/drone/drone/pkg/types"
	. "github.com/franela/goblin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

var tokenTests = []struct {
	inLabel    string
	inBody     string
	inSession  *settings.Session
	inHaveSess bool
	storeErr   error
	outCode    int
	outKind    string
}{
	{"", `{}`, nil, false, nil, 500, ""},
	{"", `{}`, &settings.Session{Secret: "Otto"}, false, sql.ErrNoRows, 500, ""},
	{"app1", `{"label": "app1"}`, &settings.Session{Secret: "Otto"}, true, nil, 200, types.TokenUser},
	{"app2", `{"label": "app2"}`, &settings.Session{Secret: "Otto"}, false, nil, 200, types.TokenUser},
}

func TestToken(t *testing.T) {
	store := new(mocks.Store)

	g := Goblin(t)
	g.Describe("Token", func() {
		g.It("should create tokens", func() {
			for _, test := range tokenTests {
				rw := recorder.New()
				ctx := gin.Context{Engine: gin.Default(), Writer: rw}
				body := bytes.NewBufferString(test.inBody)
				ctx.Request, _ = http.NewRequest("POST", "/api/user/tokens", body)

				ctx.Set("datastore", store)
				ctx.Set("user", &types.User{Login: "Freya"})

				config := settings.Settings{Session: test.inSession}
				ctx.Set("settings", &config)
				if test.inSession != nil {
					// only set these up if we've got Session configuration
					if test.inHaveSess {
						ctx.Set("session", session.New(test.inSession))
					}

					// prepare the mock datastore
					store.On("AddToken", mock.AnythingOfType("*types.Token")).Return(test.storeErr).Once()
				}

				PostToken(&ctx)

				g.Assert(rw.Code).Equal(test.outCode)
				if test.outCode != 200 {
					continue
				}

				var respjson map[string]interface{}
				json.Unmarshal(rw.Body.Bytes(), &respjson)
				g.Assert(respjson["kind"]).Equal(types.TokenUser)
				g.Assert(respjson["label"]).Equal(test.inLabel)

				// this is probably going too far... maybe just validate hash is not empty?
				jwt.Parse(respjson["hash"].(string), func(token *jwt.Token) (interface{}, error) {
					_, ok := token.Method.(*jwt.SigningMethodHMAC)
					g.Assert(ok).IsTrue()
					g.Assert(token.Claims["label"]).Equal(test.inLabel)
					return nil, nil
				})
			}
		})

		g.It("should delete tokens")
	})
}
