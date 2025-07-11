package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"gh.tarampamp.am/video-dl-bot/internal/bot"
	"gh.tarampamp.am/video-dl-bot/internal/cli/cmd"
	"gh.tarampamp.am/video-dl-bot/internal/logger"
	"gh.tarampamp.am/video-dl-bot/internal/version"
)

//go:generate go run ./generate/readme.go

// App represents the CLI application structure.
type App struct {
	cmd cmd.Command
	opt struct {
		PidFile       string
		DoHealthcheck bool

		BotToken               string
		CookiesFile            string
		MaxConcurrentDownloads uint
	}
}

// NewApp initializes a new CLI application instance.
func NewApp(name string) *App { //nolint:funlen,gocognit,gocyclo
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
		logLevelFlag = cmd.Flag[string]{
			Names:   []string{"log-level"},
			Usage:   "Logging level (" + strings.Join(logger.LevelStrings(), "/") + ")",
			EnvVars: []string{"LOG_LEVEL"},
			Default: logger.InfoLevel.String(),
			Validator: func(_ *cmd.Command, v string) error {
				if _, err := logger.ParseLevel(v); err != nil {
					return fmt.Errorf("invalid log level: %w", err)
				}

				return nil
			},
		}
		logFormatFlag = cmd.Flag[string]{
			Names:   []string{"log-format"},
			Usage:   "Logging format (" + strings.Join(logger.FormatStrings(), "/") + ")",
			EnvVars: []string{"LOG_FORMAT"},
			Default: logger.ConsoleFormat.String(),
			Validator: func(_ *cmd.Command, v string) error {
				if _, err := logger.ParseFormat(v); err != nil {
					return fmt.Errorf("invalid log format: %w", err)
				}

				return nil
			},
		}
		botTokenFlag = cmd.Flag[string]{
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
		cookiesFileFlag = cmd.Flag[string]{
			Names:   []string{"cookies-file", "c"},
			Usage:   "Path to the file with cookies (netscape-formatted) for the bot (optional)",
			EnvVars: []string{"COOKIES_FILE"},
			Default: app.opt.CookiesFile,
			Validator: func(_ *cmd.Command, v string) error {
				if v != "" {
					if stat, err := os.Stat(v); err != nil {
						return fmt.Errorf("failed to access cookies file: %w", err)
					} else if stat.IsDir() {
						return fmt.Errorf("cookies file path cannot be a directory")
					}
				}

				return nil
			},
		}
		maxConcurrentDownloadsFlag = cmd.Flag[uint]{
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
		pidFileFlag = cmd.Flag[string]{
			Names:   []string{"pid-file"},
			Usage:   "Path to the file where the process ID will be stored",
			EnvVars: []string{"PID_FILE"},
			Default: app.opt.PidFile,
			Validator: func(_ *cmd.Command, _ string) error {
				if app.opt.PidFile == "" {
					return nil
				}

				if _, err := os.Stat(app.opt.PidFile); err != nil {
					if os.IsNotExist(err) {
						return nil // it's okay - file shouldn't exist before the bot starts
					}
				}

				return errors.New("pid file path already exists (another instance may be running), or the path is invalid")
			},
		}
		healthcheckFlag = cmd.Flag[bool]{
			Names: []string{"healthcheck"},
			Usage: "Check the health of the bot (useful for Docker/K8s healthcheck; pid file must be set) and exit",
		}
	)

	app.cmd.Flags = []cmd.Flagger{
		&logLevelFlag,
		&logFormatFlag,
		&botTokenFlag,
		&cookiesFileFlag,
		&maxConcurrentDownloadsFlag,
		&pidFileFlag,
		&healthcheckFlag,
	}

	// define main command action
	app.cmd.Action = func(ctx context.Context, c *cmd.Command, args []string) error {
		var (
			logLevel, _  = logger.ParseLevel(*logLevelFlag.Value)   // error ignored because the flag validates itself
			logFormat, _ = logger.ParseFormat(*logFormatFlag.Value) // --//--
		)

		log, logErr := logger.New(logLevel, logFormat) // create new logger instance
		if logErr != nil {
			return logErr
		}

		setIfFlagIsSet(&app.opt.PidFile, pidFileFlag)
		setIfFlagIsSet(&app.opt.DoHealthcheck, healthcheckFlag)
		setIfFlagIsSet(&app.opt.BotToken, botTokenFlag)
		setIfFlagIsSet(&app.opt.CookiesFile, cookiesFileFlag)
		setIfFlagIsSet(&app.opt.MaxConcurrentDownloads, maxConcurrentDownloadsFlag)

		if app.opt.DoHealthcheck {
			if app.opt.PidFile == "" {
				return errors.New("pid file must be set for healthcheck")
			}

			b, err := os.ReadFile(app.opt.PidFile)
			if err != nil {
				return fmt.Errorf("failed to read pid file: %w", err)
			}

			pid, err := strconv.Atoi(string(b))
			if err != nil {
				return fmt.Errorf("invalid pid in file %s: %w", app.opt.PidFile, err)
			}

			// check if process is alive. if sig is 0, then no signal is sent, but error checking is still per‐
			// formed; this can be used to check for the existence of a process ID or process group ID
			if err = syscall.Kill(pid, syscall.Signal(0)); err != nil {
				return errors.New("process is not running")
			}

			log.Info("healthcheck successful", slog.Int("pid", pid), slog.String("pid_file", app.opt.PidFile))

			return nil // healthcheck successful
		}

		if app.opt.PidFile != "" {
			// the file shouldn't exist before the bot starts, so we check if it exists
			if _, err := os.Stat(app.opt.PidFile); err == nil {
				return fmt.Errorf("pid file already exists: %s (another instance may be running)", app.opt.PidFile)
			}

			// write the PID to the specified file
			if err := os.WriteFile(app.opt.PidFile, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil { //nolint:gosec,mnd
				return fmt.Errorf("failed to write PID file: %w", err)
			}

			log.Info("pid file created", "path", app.opt.PidFile)

			defer func() { _ = os.Remove(app.opt.PidFile) }() // remove PID file on exit
		}

		if app.opt.CookiesFile != "" {
			// Copy the file with cookies if it is set through environment variables, to
			// avoid issues with read-only mounted secrets like this one:
			//
			// File \"/usr/bin/yt-dlp/__main__.py\", line 17, in <module>;
			// ...
			// with open(file, 'w' if write else 'r', encoding='utf-8')
			// OSError: [Errno 30] Read-only file system: '/cookies.txt'
			content, rErr := os.ReadFile(app.opt.CookiesFile)
			if rErr != nil {
				return fmt.Errorf("failed to read cookies file: %w", rErr)
			}

			tmpDir, tmpDirErr := os.MkdirTemp("", "cookies-*")
			if tmpDirErr != nil {
				return fmt.Errorf("failed to create temporary directory for cookies: %w", tmpDirErr)
			}

			defer func() { _ = os.RemoveAll(tmpDir) }()

			tmpCookiesFile := filepath.Join(tmpDir, "cookies.txt")

			if err := os.WriteFile(tmpCookiesFile, content, 0o600); err != nil { //nolint:mnd
				return err
			}

			app.opt.CookiesFile = tmpCookiesFile
		}

		return app.run(ctx, log)
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
func (a *App) run(ctx context.Context, log *slog.Logger) error {
	var botOpts = []bot.Option{
		bot.WithLogger(log.With("source", "telebot")),
		bot.WithMaxConcurrentDownloads(a.opt.MaxConcurrentDownloads),
	}

	if a.opt.CookiesFile != "" {
		botOpts = append(botOpts, bot.WithCookiesFile(a.opt.CookiesFile))
	} else {
		log.Warn("no cookies file provided, some sites may not work without it")
	}

	b, err := bot.NewBot(ctx, a.opt.BotToken, botOpts...)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	log.Info("starting bot")

	b.Start(ctx) // blocking call

	log.Info("bot stopped")

	return nil
}
