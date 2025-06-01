package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"gh.tarampamp.am/video-dl-bot/internal/cli/cmd"
	"gh.tarampamp.am/video-dl-bot/internal/version"
	ytdlp "gh.tarampamp.am/video-dl-bot/internal/yt-dlp"
)

//go:generate go run ./generate/readme.go

type App struct {
	cmd cmd.Command
	opt struct {
		BotToken string
	}
}

func NewApp(name string) *App {
	var app = App{
		cmd: cmd.Command{
			Name:        name,
			Description: "This is a video download bot that allows you to download videos not leaving Telegram.",
			Version:     version.Version(),
		},
	}

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
	)

	app.cmd.Flags = []cmd.Flagger{
		&botToken,
	}

	app.cmd.Action = func(ctx context.Context, c *cmd.Command, args []string) error {
		setIfFlagIsSet(&app.opt.BotToken, botToken)

		return app.run(ctx)
	}

	return &app
}

// setIfFlagIsSet sets the value from the flag to the option if the flag is set and the value is not nil.
func setIfFlagIsSet[T cmd.FlagType](target *T, source cmd.Flag[T]) {
	if target == nil || source.Value == nil || !source.IsSet() {
		return
	}

	*target = *source.Value
}

// Run runs the application.
func (a *App) Run(ctx context.Context, args []string) error { return a.cmd.Run(ctx, args) }

// Help returns the help message.
func (a *App) Help() string { return a.cmd.Help() }

// run in the main logic of the application.
func (a *App) run(ctx context.Context) error {
	bot, botErr := tele.NewBot(tele.Settings{
		Token:  a.opt.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if botErr != nil {
		return botErr
	}

	for _, event := range [...]string{tele.OnText, tele.OnForward, tele.OnReply} {
		bot.Handle(event, a.handleVideoDownload(ctx))
	}

	// Shutdown logic in a goroutine
	go func() {
		<-ctx.Done()
		fmt.Println("Context cancelled, shutting down the bot")
		bot.Stop()
	}()

	bot.Start() // blocking call

	fmt.Println("Bot has stopped")

	return nil
}

// rmHashtagsRe is a regular expression to remove hashtags from video titles (including cyrillic hashtags).
var rmHashtagsRe = regexp.MustCompile(`(?i)[#＃][\p{L}\p{N}_-]+`)

// define a global semaphore to limit the number of concurrent downloads
var downloadSemaphore = make(chan struct{}, 5) // limit to 5 concurrent downloads

func (a *App) handleVideoDownload(ctx context.Context) tele.HandlerFunc {
	return func(c tele.Context) error {
		select {
		case downloadSemaphore <- struct{}{}: // acquire a semaphore slot
		case <-ctx.Done():
			return ctx.Err() // if context is cancelled, return immediately
		}

		defer func() { <-downloadSemaphore }() // release the semaphore slot when done

		var (
			bot     = c.Bot()
			user    = c.Sender()
			userMsg = c.Message()
			userUrl = strings.Trim(c.Text(), " \n\r\t")
		)

		if u, uErr := url.Parse(userUrl); uErr != nil || u.Scheme == "" || u.Host == "" {
			_, err := bot.Reply(userMsg, "☠ Please provide a valid video URL.")

			return err
		}

		replyMsg, replyErr := bot.Reply(userMsg, "😉 Processing your video download request. Please, be patient")
		if replyErr != nil {
			return replyErr
		}

		dl, dlErr := ytdlp.Download(ctx, userUrl)
		if dlErr != nil {
			_, err := bot.Edit(replyMsg, fmt.Sprintf("❌ Failed to download video: %s", dlErr.Error()))

			return err
		}

		stat, statErr := os.Stat(dl.Filepath)
		if statErr != nil {
			_, err := bot.Edit(replyMsg, fmt.Sprintf("❌ Downloaded video file not available: %s", statErr.Error()))

			return err
		}

		defer func() { _ = os.Remove(dl.Filepath) }() // clean up the downloaded file after sending

		if _, err := bot.Edit(replyMsg, "😇 Video download completed successfully! Now sending it to you..."); err != nil {
			return err
		}

		var text strings.Builder

		if dl.Title != "" {
			text.WriteString(strings.TrimSpace(rmHashtagsRe.ReplaceAllString(dl.Title, " ")))
		}

		if dl.WebpageURL != "" {
			text.WriteString(" // [")

			if dl.Extractor != "" {
				text.WriteString(dl.Extractor)
			} else {
				text.WriteString("source")
			}

			text.WriteString("](")
			text.WriteString(dl.WebpageURL)
			text.WriteString(")")
		}

		var attachment any

		if stat.Size() <= 50*1024*1024 {
			attachment = &tele.Video{ // send a video as-is if file size less than or equal to 50MB
				File:    tele.FromDisk(dl.Filepath),
				Caption: text.String(),
			}
		} else {
			attachment = &tele.Document{ // otherwise send it as a document
				File:    tele.FromDisk(dl.Filepath),
				Caption: text.String(),
			}
		}

		if _, err := bot.Send(user, attachment, &tele.SendOptions{
			ParseMode:             tele.ModeMarkdown,
			DisableWebPagePreview: true,
		}); err != nil {
			_, err = bot.Edit(
				replyMsg,
				fmt.Sprintf("❌ Failed to send video (%d Mb): %s", stat.Size()/1024/1024, err.Error()),
			)

			return err
		}

		// delete the original message
		_ = bot.Delete(userMsg)

		// and the reply to keep the chat clean
		return bot.Delete(replyMsg)
	}
}
