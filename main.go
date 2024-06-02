package main

import (
	"github.com/ezshare/server/auth"
	"github.com/ezshare/server/config"
	"github.com/ezshare/server/router"
	"github.com/ezshare/server/server"
)

func main() {
	cf := config.LoadConfig()
	users, _ := auth.LoadUsersFile(cf.UsersFile, cf.Secret, cf.SessionTimeoutSeconds)
	mux := router.Router(*cf, users)
	_ = server.Start(mux, cf.ServerAddress, "", "")
}
