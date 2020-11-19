package stats

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/xjasonlyu/tun2socks/log"
	"github.com/xjasonlyu/tun2socks/tunnel"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/gorilla/websocket"
	"github.com/urfave/cli/v2"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func Start(addr, secret string, app *cli.App) error {
	r := chi.NewRouter()

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		MaxAge:         300,
	})

	r.Use(c.Handler)

	hello := func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, render.M{"hello": app.Name})
	}

	version := func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, render.M{"version": app.Version})
	}

	r.Group(func(r chi.Router) {
		r.Use(authenticator(secret))
		r.Get("/", hello)
		r.Get("/version", version)
		r.Get("/logs", getLogs)
		r.Get("/traffic", traffic)
		r.Mount("/connections", connectionRouter())
	})

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}

	go func() {
		_ = http.Serve(listener, r)
	}()

	return nil
}

func authenticator(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Browser websocket not support custom header
			if websocket.IsWebSocketUpgrade(r) && r.URL.Query().Get("token") != "" {
				token := r.URL.Query().Get("token")
				if token != secret {
					render.Status(r, http.StatusUnauthorized)
					render.JSON(w, r, ErrUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			header := r.Header.Get("Authorization")
			text := strings.SplitN(header, " ", 2)

			hasInvalidHeader := text[0] != "Bearer"
			hasInvalidSecret := len(text) != 2 || text[1] != secret
			if hasInvalidHeader || hasInvalidSecret {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, ErrUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

func getLogs(w http.ResponseWriter, r *http.Request) {
	lvl := r.URL.Query().Get("level")
	if lvl == "" {
		lvl = "info" /* default */
	}

	level, err := log.ParseLevel(lvl)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, ErrBadRequest)
		return
	}

	var wsConn *websocket.Conn
	if websocket.IsWebSocketUpgrade(r) {
		wsConn, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
	}

	if wsConn == nil {
		w.Header().Set("Content-Type", "application/json")
		render.Status(r, http.StatusOK)
	}

	sub := log.Subscribe()
	defer log.UnSubscribe(sub)

	buf := &bytes.Buffer{}
	for elm := range sub {
		buf.Reset()

		e := elm.(*log.Event)
		if e.Level > level {
			continue
		}

		if err := json.NewEncoder(buf).Encode(e); err != nil {
			break
		}

		if wsConn == nil {
			_, err = w.Write(buf.Bytes())
			w.(http.Flusher).Flush()
		} else {
			err = wsConn.WriteMessage(websocket.TextMessage, buf.Bytes())
		}

		if err != nil {
			break
		}
	}
}

type Traffic struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

func traffic(w http.ResponseWriter, r *http.Request) {
	var wsConn *websocket.Conn
	if websocket.IsWebSocketUpgrade(r) {
		var err error
		wsConn, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
	}

	if wsConn == nil {
		w.Header().Set("Content-Type", "application/json")
		render.Status(r, http.StatusOK)
	}

	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	t := tunnel.DefaultManager
	buf := &bytes.Buffer{}
	var err error
	for range tick.C {
		buf.Reset()
		up, down := t.Now()
		if err := json.NewEncoder(buf).Encode(Traffic{
			Up:   up,
			Down: down,
		}); err != nil {
			break
		}

		if wsConn == nil {
			_, err = w.Write(buf.Bytes())
			w.(http.Flusher).Flush()
		} else {
			err = wsConn.WriteMessage(websocket.TextMessage, buf.Bytes())
		}

		if err != nil {
			break
		}
	}
}
