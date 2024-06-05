package router

import (
	"encoding/json"
	"github.com/ezshare/server/auth"
	"github.com/ezshare/server/config"
	"github.com/ezshare/server/ui"
	"github.com/ezshare/server/ws"
	"github.com/gorilla/handlers"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)
import "github.com/gorilla/mux"

type UIConfig struct {
	User                     string `json:"user"`
	LoggedIn                 bool   `json:"loggedIn"`
	RoomName                 string `json:"roomName"`
	CloseRoomWhenOwnerLeaves bool   `json:"closeRoomWhenOwnerLeaves"`
}

func responseLogger(r *http.Request, status, size int, duration time.Duration) {
	log.Info().Str("host", r.Host).Str("method", r.Method).Str("path", r.URL.Path).Str("ip", r.RemoteAddr).Int("status", status).Int("size", size).Dur("duration", duration).Msg("response")
}

func Router(config config.Config, rooms *ws.Rooms, users *auth.Users) *mux.Router {
	// 1. 创建一个新的路由对象
	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseLogger(r, http.StatusNotFound, 0, 0)
	})

	// 2. 配置路由中间件
	router.Use(handlers.CORS(handlers.AllowedMethods([]string{"GET", "POST"}), handlers.AllowedOriginValidator(config.CheckOrigin)))
	router.Use(hlog.AccessHandler(responseLogger))

	// 3. 配置路由
	router.HandleFunc("/stream", rooms.Upgrade)
	router.Methods("POST").Path("/login").HandlerFunc(users.Authenticate)
	router.Methods("POST").Path("/logout").HandlerFunc(users.Logout)
	router.Methods("GET").Path("/config").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, loggedIn := users.CurrentUser(r)
		_ = json.NewEncoder(w).Encode(&UIConfig{
			User:                     user,
			LoggedIn:                 loggedIn,
			RoomName:                 rooms.RandRoomName(),
			CloseRoomWhenOwnerLeaves: config.CloseRoomWhenOwnerLeaves,
		})
	})
	router.Methods("POST").Path("/login").HandlerFunc(users.Authenticate)
	router.Methods("POST").Path("/logout").HandlerFunc(users.Logout)
	ui.Register(router)
	return router
}
