package cmd

import (
	"github.com/ezshare/server/auth"
	"github.com/ezshare/server/config"
	"github.com/ezshare/server/router"
	"github.com/ezshare/server/server"
	"github.com/ezshare/server/turn"
	"github.com/ezshare/server/ws"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
)

func Start(ctx *cli.Context) {
	c, err := config.LoadConfig()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load config")
		return
	}

	users, err := auth.LoadUsersFile(c.UsersFile, c.Secret, c.SessionTimeoutSeconds)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load users file")
		return
	}

	turnServer, err := turn.Start(c)
	if err != nil {
		log.Error().Err(err).Msg("Could not start turn server")
		return
	}

	rooms := ws.NewRooms(turnServer, users, *c)
	go rooms.Start()

	r := router.Router(*c, rooms, users)
	err = server.Start(r, c.ServerAddress, c.TLSCertFile, c.TLSKeyFile)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start http server")
		return
	}

	log.Info().Msg("ezshare started")
	return
}
