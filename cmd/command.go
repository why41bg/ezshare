package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli"
	"os"
)

func Run() {
	app := cli.NewApp()
	app.Name = "ezshare"
	app.Usage = "ezshare is a simple screen share tool"
	app.Commands = []cli.Command{
		{
			Name:    "start",
			Usage:   "Start the ezshare server",
			Aliases: []string{"s"},
			Action:  Start,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Err(err).Msg("app error")
	}

}
