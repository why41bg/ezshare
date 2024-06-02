package router

import (
	"encoding/json"
	"github.com/ezshare/server/auth"
	"github.com/ezshare/server/config"
	"github.com/ezshare/server/ui"
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

func Router(config config.Config, users *auth.Users) *mux.Router {
	router := mux.NewRouter()
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseLogger(r, http.StatusNotFound, 0, 0)
	})

	// 对于每个到来的请求，先执行 CORS 策略，然后再交给具体的处理函数，最后还需要进行日志记录
	router.Use(handlers.CORS(handlers.AllowedMethods([]string{"GET", "POST"}), handlers.AllowedOriginValidator(config.CheckOrigin)))
	router.Use(hlog.AccessHandler(responseLogger))

	// 具体的路由规则
	router.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {}) // TODO: 实现 stream
	router.Methods("GET").Path("/config").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, loggedIn := users.CurrentUser(r)
		_ = json.NewEncoder(w).Encode(&UIConfig{
			User:                     user,
			LoggedIn:                 loggedIn,
			RoomName:                 "default", // TODO: 随机生成一个房间名
			CloseRoomWhenOwnerLeaves: config.CloseRoomWhenOwnerLeaves,
		})
	})
	router.Methods("POST").Path("/login").HandlerFunc(users.Authenticate)
	router.Methods("POST").Path("/logout").HandlerFunc(users.Logout)
	ui.Register(router)
	return router
}
