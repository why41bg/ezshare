package main

import (
	"github.com/ezshare/server/auth"
	"github.com/ezshare/server/config"
	"github.com/ezshare/server/router"
	"github.com/ezshare/server/server"
	"github.com/ezshare/server/turn"
)

func main() {
	cf := config.LoadConfig()
	users, _ := auth.LoadUsersFile(cf.UsersFile, cf.Secret, cf.SessionTimeoutSeconds)
	mux := router.Router(*cf, users)
	_, _ = turn.Start(cf)
	_ = server.Start(mux, cf.ServerAddress, "", "")
}
