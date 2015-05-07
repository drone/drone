package server

import (
	"bufio"
	"net/http"
	"strconv"
	"time"

	"github.com/drone/drone/eventbus"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	// "github.com/koding/websocketproxy"
)

const (
	// Time allowed to write the message to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// GetRepoEvents will upgrade the connection to a Websocket and will stream
// event updates to the browser.
func GetRepoEvents(c *gin.Context) {
	bus := ToBus(c)
	repo := ToRepo(c)

	// upgrade the websocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.Fail(400, err)
		return
	}

	ticker := time.NewTicker(pingPeriod)
	eventc := make(chan *eventbus.Event)
	bus.Subscribe(eventc)
	defer func() {
		bus.Unsubscribe(eventc)
		ticker.Stop()
		ws.Close()
		close(eventc)
		log.Infof("closed websocket")
	}()

	go func() {
		for {
			select {
			case <-c.Writer.CloseNotify():
				ws.Close()
				return
			case event := <-eventc:
				if event == nil {
					log.Infof("closed websocket")
					ws.Close()
					return
				}
				if event.Kind == eventbus.EventRepo && event.Name == repo.FullName {
					ws.WriteMessage(websocket.TextMessage, event.Msg)
					break
				}
			case <-ticker.C:
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					log.Infof("closed websocket")
					ws.Close()
					return
				}
			}
		}
	}()

	readWebsocket(ws)
}

func GetStream(c *gin.Context) {
	// store := ToDatastore(c)
	repo := ToRepo(c)
	runner := ToRunner(c)
	build, _ := strconv.Atoi(c.Params.ByName("build"))
	task, _ := strconv.Atoi(c.Params.ByName("number"))

	// agent, err := store.BuildAgent(repo.FullName, build)
	// if err != nil {
	// 	c.Fail(404, err)
	// 	return
	// }

	rc, err := runner.Logs(repo.FullName, build, task)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// upgrade the websocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.Fail(400, err)
		return
	}

	var ticker = time.NewTicker(pingPeriod)
	var out = make(chan []byte)
	defer func() {
		log.Infof("closed stdout websocket")
		ticker.Stop()
		rc.Close()
		ws.Close()
	}()

	go func() {
		for {
			select {
			case <-c.Writer.CloseNotify():
				rc.Close()
				ws.Close()
				return
			case line := <-out:
				ws.WriteMessage(websocket.TextMessage, line)
			case <-ticker.C:
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					rc.Close()
					ws.Close()
					return
				}
			}
		}
	}()

	go func() {
		rd := bufio.NewReader(rc)
		for {
			str, err := rd.ReadBytes('\n')

			if err != nil {
				break
			}
			if len(str) == 0 {
				break
			}

			out <- str
		}
		rc.Close()
		ws.Close()
	}()

	readWebsocket(ws)

	// url_, err := url.Parse("ws://" + agent.Addr)
	// if err != nil {
	// 	c.Fail(500, err)
	// 	return
	// }
	// url_.Path = fmt.Sprintf("/stream/%s/%v/%v", repo.FullName, build, task)
	// proxy := websocketproxy.NewProxy(url_)
	// proxy.ServeHTTP(c.Writer, c.Request)

	// log.Debugf("closed websocket")
}

// readWebsocket will block while reading the websocket data
func readWebsocket(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}
