package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"gh.tarampamp.am/video-dl-bot/internal/cli/cmd"
	"gh.tarampamp.am/video-dl-bot/internal/filestorage"
	"gh.tarampamp.am/video-dl-bot/internal/version"
	ytdlp "gh.tarampamp.am/video-dl-bot/internal/yt-dlp"
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
	bot, botErr := tele.NewBot(tele.Settings{
		Token:  a.opt.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second}, //nolint:mnd // value taken from the official example
	})
	if botErr != nil {
		return botErr
	}

	// handle /start command
	bot.Handle("/start", func(c tele.Context) error {
		return reply(bot, c.Message(), fmt.Sprintf(`Hello %s! I can help you download videos from the internet.

Send me a video URL or forward a message with a video to download it.`,
			c.Sender().FirstName,
		))
	})

	// semaphore to control max concurrent downloads
	var lim = make(semaphore, a.opt.MaxConcurrentDownloads)

	// register handlers for different types of messages
	for _, event := range [...]string{tele.OnText, tele.OnForward, tele.OnReply} {
		bot.Handle(event, a.handleVideoDownload(ctx, lim))
	}

	// graceful shutdown on context cancellation
	go func() {
		<-ctx.Done()
		fmt.Println("Context cancelled, shutting down the bot") //nolint:forbidigo // TODO: implement a proper logger

		bot.Stop()
	}()

	bot.Start() // blocking call to start receiving updates

	fmt.Println("Bot has stopped") //nolint:forbidigo // TODO: implement a proper logger

	return nil
}

// semaphore is a typed channel for limiting concurrency.
type semaphore chan struct{}

// Release frees up a slot in the semaphore.
func (s semaphore) Release() { <-s }

// Acquire attempts to occupy a semaphore slot or returns if the context is cancelled.
func (s semaphore) Acquire(ctx context.Context) error {
	select {
	case s <- struct{}{}: // acquire a semaphore slot
		if err := ctx.Err(); err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// handleVideoDownload returns a handler for processing video download requests.
func (a *App) handleVideoDownload(pCtx context.Context, lim semaphore) tele.HandlerFunc { //nolint:funlen
	return func(c tele.Context) error {
		ctx, cancel := context.WithCancel(pCtx)
		defer cancel()

		if err := lim.Acquire(ctx); err != nil {
			return err
		}
		defer lim.Release()

		var (
			bot, user        = c.Bot(), c.Sender()
			userMsg, userUrl = c.Message(), strings.Trim(c.Text(), " \n\r\t")
		)

		fmt.Printf( //nolint:forbidigo // TODO: implement a proper logger
			"Video downloading request from user %s (id:%d), url: %s\n",
			user.FirstName,
			user.ID,
			userUrl,
		)

		// validate the user-supplied URL
		if u, uErr := url.Parse(userUrl); uErr != nil || u.Scheme == "" || u.Host == "" {
			_ = bot.React(user, userMsg, tele.Reactions{
				Reactions: []tele.Reaction{{Type: tele.ReactionTypeEmoji, Emoji: "💩"}},
			})

			return reply(bot, userMsg, "❌ Please provide a valid video URL")
		}

		// clear any previous reactions once we're done
		defer func() { _ = bot.React(user, userMsg, tele.Reactions{Reactions: []tele.Reaction{}}) }()

		// react with an emoji while downloading
		_ = bot.React(user, userMsg, tele.Reactions{
			Reactions: []tele.Reaction{{Type: tele.ReactionTypeEmoji, Emoji: "🫡"}},
			Big:       true,
		})

		// show "recording video" status
		stopDownloadingAction := setChatAction(ctx, bot, user, tele.RecordingVideo)
		defer stopDownloadingAction()

		// download the video
		dl, dlErr := ytdlp.Download(ctx, userUrl)
		if dlErr != nil {
			return reply(bot, userMsg, "❌ Failed to download video: "+dlErr.Error())
		}

		stopDownloadingAction()

		// stat the file to get size info
		stat, statErr := os.Stat(dl.Filepath)
		if statErr != nil {
			return reply(bot, userMsg, "❌ Downloaded video file not available: "+statErr.Error())
		}

		defer func() { _ = os.Remove(dl.Filepath) }() // clean up the downloaded file after sending

		// open the downloaded file
		fp, fpErr := os.Open(dl.Filepath)
		if fpErr != nil {
			return reply(bot, userMsg, "❌ Failed to open downloaded video file: "+fpErr.Error())
		}

		defer func() { _ = fp.Close() }() // ensure the file is closed after sending

		// show "uploading video" status
		stopUploadingAction := setChatAction(ctx, bot, user, tele.UploadingVideo)
		defer stopUploadingAction()

		// react while uploading
		_ = bot.React(user, userMsg, tele.Reactions{
			Reactions: []tele.Reaction{{Type: tele.ReactionTypeEmoji, Emoji: "⚡"}},
			Big:       true,
		})

		// telegram upload limit is 50MB
		if stat.Size() <= 50*1024*1024 {
			if _, err := bot.Reply(userMsg, &tele.Video{File: tele.FromReader(fp)}); err != nil {
				return reply(bot, userMsg, fmt.Sprintf(
					"❌ Failed to send video (%d Mb): %s",
					stat.Size()/1024/1024, //nolint:mnd
					err.Error(),
				))
			}
		} else {
			// upload to file hosting if file is too large
			fileUrl, urlErr := filestorage.UploadToFileBin(ctx, fp, fmt.Sprintf("video%s", filepath.Ext(dl.Filepath)))
			if urlErr != nil {
				return reply(bot, userMsg, "❌ Failed to upload video to file hosting: "+urlErr.Error())
			}

			if _, err := bot.Reply(userMsg, "Your video is ready for download:", &tele.ReplyMarkup{
				ResizeKeyboard: true,
				InlineKeyboard: [][]tele.InlineButton{
					{{
						Text: "🚀 Download video (this link will expire in a couple of days)",
						URL:  fileUrl,
					}},
				},
			}); err != nil {
				return err
			}
		}

		stopUploadingAction()

		return nil
	}
}

// setChatAction displays a periodic chat action (e.g. typing, recording).
// It returns a cancel function that should be deferred to stop the action.
func setChatAction(ctx context.Context, bot tele.API, user *tele.User, action tele.ChatAction) (stop func()) {
	ctx, stop = context.WithCancel(ctx)

	go func() {
		defer stop()

		if err := bot.Notify(user, action); err != nil {
			return
		}

		ticker := time.NewTicker(5 * time.Second) //nolint:mnd // this is Telegram's recommended interval
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				if err := ctx.Err(); err != nil {
					return
				}

				if err := bot.Notify(user, action); err != nil {
					return
				}
			}
		}
	}()

	return
}

// reply sends a plain text reply to a message.
func reply(bot tele.API, to *tele.Message, msg string) (err error) {
	_, err = bot.Reply(to, msg)

	return
}
