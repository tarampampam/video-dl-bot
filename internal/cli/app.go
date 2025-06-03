package cli

import (
	"context"
	"fmt"
	"regexp"

	"gh.tarampamp.am/video-dl-bot/internal/bot"
	"gh.tarampamp.am/video-dl-bot/internal/cli/cmd"
	"gh.tarampamp.am/video-dl-bot/internal/version"
)

//go:generate go run ./generate/readme.go

// App represents the CLI application structure.
type App struct {
	cmd cmd.Command
	opt struct {
		BotToken               string
		MaxConcurrentDownloads uint
	}
}

// NewApp initializes a new CLI application instance.
func NewApp(name string) *App { //nolint:funlen
	var app = App{
		cmd: cmd.Command{
			Name:        name,
			Description: "This is a video download bot that allows you to download videos not leaving Telegram.",
			Version:     version.Version(),
		},
	}

	// set default options
	app.opt.MaxConcurrentDownloads = 5

	// define CLI flags with validation
	var (
		botToken = cmd.Flag[string]{
			Names:   []string{"bot-token", "t"},
			Usage:   "Telegram bot token",
			EnvVars: []string{"BOT_TOKEN"},
			Default: app.opt.BotToken,
			Validator: func(_ *cmd.Command, v string) error {
				if v == "" {
					return fmt.Errorf("telegram bot token is required")
				}

				if len(v) < 10 || len(v) > 100 {
					return fmt.Errorf("telegram bot token must be between 10 and 100 characters long")
				}

				if !regexp.MustCompile(`^[0-9]{8,10}:[a-zA-Z0-9_-]{35}$`).MatchString(v) {
					return fmt.Errorf("telegram bot token is invalid")
				}

				return nil
			},
		}

		maxConcurrentDownloads = cmd.Flag[uint]{
			Names:   []string{"max-concurrent-downloads", "m"},
			Usage:   "Maximum number of concurrent downloads",
			EnvVars: []string{"MAX_CONCURRENT_DOWNLOADS"},
			Default: app.opt.MaxConcurrentDownloads,
			Validator: func(_ *cmd.Command, v uint) error {
				if v < 1 || v > 100 {
					return fmt.Errorf("maximum number of concurrent downloads must be between 1 and 100")
				}

				return nil
			},
		}
	)

	app.cmd.Flags = []cmd.Flagger{
		&botToken,
		&maxConcurrentDownloads,
	}

	// define main command action
	app.cmd.Action = func(ctx context.Context, c *cmd.Command, args []string) error {
		setIfFlagIsSet(&app.opt.BotToken, botToken)
		setIfFlagIsSet(&app.opt.MaxConcurrentDownloads, maxConcurrentDownloads)

		return app.run(ctx)
	}

	return &app
}

// setIfFlagIsSet assigns a flag value to target if the flag is set and non-nil.
func setIfFlagIsSet[T cmd.FlagType](target *T, source cmd.Flag[T]) {
	if target == nil || source.Value == nil || !source.IsSet() {
		return
	}

	*target = *source.Value
}

// Run starts the CLI command execution.
func (a *App) Run(ctx context.Context, args []string) error { return a.cmd.Run(ctx, args) }

// Help returns the CLI help message.
func (a *App) Help() string { return a.cmd.Help() }

// run contains the main bot initialization and event loop.
func (a *App) run(ctx context.Context) error {
	b, err := bot.NewBot(ctx, a.opt.BotToken, a.opt.MaxConcurrentDownloads)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	b.Start(ctx) // blocking call

	return nil
}
