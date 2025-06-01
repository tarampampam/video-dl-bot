package cli

import (
	"context"
	"fmt"

	"gh.tarampamp.am/video-dl-bot/internal/cli/cmd"
	"gh.tarampamp.am/video-dl-bot/internal/version"
)

//go:generate go run ./generate/readme.go

type App struct {
	cmd cmd.Command
}

func NewApp(name string) *App {
	var app = App{
		cmd: cmd.Command{
			Name:        name,
			Description: "This is a video download bot that allows you to download videos not leaving Telegram.",
			Version:     version.Version(),
		},
	}

	app.cmd.Action = func(ctx context.Context, c *cmd.Command, args []string) error {
		return app.run(ctx)
	}

	return &app
}

// Run runs the application.
func (a *App) Run(ctx context.Context, args []string) error { return a.cmd.Run(ctx, args) }

// Help returns the help message.
func (a *App) Help() string { return a.cmd.Help() }

// run in the main logic of the application.
func (a *App) run(_ context.Context) error {
	fmt.Println("not implemented yet")

	return nil
}
