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

func Start(ctx *cli.Context) error {
	// 1. 读取配置文件
	c := config.LoadConfig()

	// 2. 加载用户信息
	users, err := auth.LoadUsersFile(c.UsersFile, c.Secret, c.SessionTimeoutSeconds)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load users file")
		return err
	}

	// 3. 启动内置TURN服务器
	turnServer, err := turn.Start(c)
	if err != nil {
		log.Error().Err(err).Msg("Could not start turn server")
		return err
	}

	// 4. 开启一个goroutine，持续监听并处理来自Rooms的消息
	rooms := ws.NewRooms(turnServer, users, *c)
	go rooms.Start()

	// 5. 配置路由，启动HTTP服务器
	r := router.Router(*c, rooms, users)
	err = server.Start(r, c.ServerAddress, "", "")
	if err != nil {
		log.Error().Err(err).Msg("Failed to start http server")
		return err
	}
	return nil
}
